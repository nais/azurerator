package secrets

import (
	"fmt"

	"github.com/nais/liberator/pkg/kubernetes"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/util/crypto"
)

type Extractor struct {
	secretLists kubernetes.SecretLists
	keys        SecretDataKeys
}

func NewExtractor(secretLists kubernetes.SecretLists, keys SecretDataKeys) *Extractor {
	return &Extractor{secretLists: secretLists, keys: keys}
}

func (e Extractor) GetKeyIdsInUse() credentials.KeyIdsInUse {
	passwordIds := make([]string, 0)
	certificateIds := make([]string, 0)

	for _, sec := range e.secretLists.Used.Items {
		certificateId := string(sec.Data[e.keys.CurrentCredentials.CertificateKeyId])

		if len(certificateId) > 0 {
			certificateIds = append(certificateIds, certificateId)
		}

		passwordId := string(sec.Data[e.keys.CurrentCredentials.PasswordKeyId])

		if len(passwordId) > 0 {
			passwordIds = append(passwordIds, passwordId)
		}
	}
	return credentials.KeyIdsInUse{
		Certificate: certificateIds,
		Password:    passwordIds,
	}
}

// GetPreviousCredentialsSet extracts the previous (if any) credential set from all the secrets matching this AzureAdApplication.
// Looks for and attempts to extract credentials matching the provided secretName parameter, otherwise falls back to
// extracting credentials from the latest in-use secret (if any).
// Ultimately returns (nil, false, nil) if no secrets match the above or if any matching secret does not contain the expected keys.
func (e Extractor) GetPreviousCredentialsSet(secretName string) (*credentials.Set, bool, error) {
	if len(secretName) == 0 {
		return nil, false, nil
	}

	allSecrets := append(e.secretLists.Unused.Items, e.secretLists.Used.Items...)

	for _, secret := range allSecrets {
		if secret.Name == secretName {
			return e.extractCredentialsSetFromSecret(secret)
		}
	}

	return nil, false, nil
}

func (e Extractor) extractCredentialsSetFromSecret(secret corev1.Secret) (*credentials.Set, bool, error) {
	currentCredential, valid, err := e.extractCurrentCredentials(secret)
	if err != nil {
		return nil, valid, fmt.Errorf("extracting current credentials set from secret: %w", err)
	}

	if !valid {
		return nil, false, nil
	}

	nextCredential, valid, err := e.extractNextCredentials(secret)
	if err != nil {
		return nil, valid, fmt.Errorf("extracting next credentials set from secret: %w", err)
	}

	if !valid {
		return nil, false, nil
	}

	return &credentials.Set{
		Current: *currentCredential,
		Next:    *nextCredential,
	}, valid, nil
}

type extractCredentialsKeys struct {
	certificateIdKey string
	clientSecretKey  string
	jwkSecretKey     string
	passwordIdKey    string
}

func (e Extractor) extractCurrentCredentials(secret corev1.Secret) (*credentials.Credentials, bool, error) {
	keys := extractCredentialsKeys{
		certificateIdKey: e.keys.CurrentCredentials.CertificateKeyId,
		clientSecretKey:  e.keys.CurrentCredentials.ClientSecret,
		jwkSecretKey:     e.keys.CurrentCredentials.Jwk,
		passwordIdKey:    e.keys.CurrentCredentials.PasswordKeyId,
	}

	return extractCredentials(secret, keys)
}

func (e Extractor) extractNextCredentials(secret corev1.Secret) (*credentials.Credentials, bool, error) {
	keys := extractCredentialsKeys{
		certificateIdKey: e.keys.NextCredentials.CertificateKeyId,
		clientSecretKey:  e.keys.NextCredentials.ClientSecret,
		jwkSecretKey:     e.keys.NextCredentials.Jwk,
		passwordIdKey:    e.keys.NextCredentials.PasswordKeyId,
	}

	return extractCredentials(secret, keys)
}

func extractCredentials(secret corev1.Secret, keys extractCredentialsKeys) (*credentials.Credentials, bool, error) {
	var clientJwk jose.JSONWebKey
	var err error

	clientSecret, exists := secret.Data[keys.clientSecretKey]
	if !isValidSecretData(clientSecret, exists) {
		return nil, false, nil
	}

	passwordId, exists := secret.Data[keys.passwordIdKey]
	if !isValidSecretData(passwordId, exists) {
		return nil, false, nil
	}

	jwkSecret, exists := secret.Data[keys.jwkSecretKey]
	if !isValidSecretData(jwkSecret, exists) {
		return nil, false, nil
	}

	certificateId, exists := secret.Data[keys.certificateIdKey]
	if !isValidSecretData(certificateId, exists) {
		return nil, false, nil
	}

	err = clientJwk.UnmarshalJSON(jwkSecret)
	if err != nil {
		return nil, false, err
	}

	return &credentials.Credentials{
		Certificate: credentials.Certificate{
			KeyId: string(certificateId),
			Jwk:   crypto.FromJwk(clientJwk),
		},
		Password: credentials.Password{
			KeyId:        string(passwordId),
			ClientSecret: string(clientSecret),
		},
	}, true, nil
}

func isValidSecretData(data []byte, found bool) bool {
	if !found || len(data) == 0 {
		return false
	}

	return true
}
