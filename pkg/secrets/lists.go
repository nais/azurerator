package secrets

import (
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/nais/liberator/pkg/kubernetes"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ExtractKeyIdsInUse(secretLists kubernetes.SecretLists) azure.KeyIdsInUse {
	passwordIds := make([]string, 0)
	certificateIds := make([]string, 0)

	for _, sec := range secretLists.Used.Items {
		certificateId := string(sec.Data[CertificateIdKey])

		if len(certificateId) > 0 {
			certificateIds = append(certificateIds, certificateId)
		}

		passwordId := string(sec.Data[PasswordIdKey])

		if len(passwordId) > 0 {
			passwordIds = append(passwordIds, passwordId)
		}
	}
	return azure.KeyIdsInUse{
		Certificate: certificateIds,
		Password:    passwordIds,
	}
}

// Extract the previous (if any) credential set from all the secrets matching this AzureAdApplication.
// Looks for and attempts to extract credentials matching the provided secretName parameter, otherwise falls back to
// extracting credentials from the latest in-use secret (if any).
// Ultimately returns (nil, false, nil) if no secrets match the above or if any matching secret does not contain the expected keys.
func ExtractCredentialsSetFromSecretLists(secretLists kubernetes.SecretLists, secretName string) (*azure.CredentialsSet, bool, error) {
	allSecrets := append(secretLists.Unused.Items, secretLists.Used.Items...)

	for _, secret := range allSecrets {
		if secret.Name == secretName {
			return extractCredentialsSetFromSecret(secret)
		}
	}

	return extractCredentialsSetFromLatestSecret(secretLists.Used.Items)
}

func extractCredentialsSetFromLatestSecret(secrets []corev1.Secret) (*azure.CredentialsSet, bool, error) {
	var latestSecret *corev1.Secret
	var latestSecretCreationTimestamp metav1.Time

	for i, secret := range secrets {
		if latestSecret == nil {
			latestSecret = &secrets[i]
			latestSecretCreationTimestamp = latestSecret.GetCreationTimestamp()
			continue
		}

		secretCreationTimestamp := secret.GetCreationTimestamp()

		if secretCreationTimestamp.After(latestSecretCreationTimestamp.Time) {
			latestSecret = &secrets[i]
			latestSecretCreationTimestamp = secretCreationTimestamp
		}
	}

	if latestSecret != nil {
		return extractCredentialsSetFromSecret(*latestSecret)
	}

	return nil, false, nil
}

func extractCredentialsSetFromSecret(secret corev1.Secret) (*azure.CredentialsSet, bool, error) {
	currentCredential, valid, err := extractCurrentcredentials(secret)
	if err != nil {
		return nil, valid, fmt.Errorf("extracting current credentials set from secret: %w", err)
	}

	nextCredential, valid, err := extractNextCredentials(secret)
	if err != nil {
		return nil, valid, fmt.Errorf("extracting next credentials set from secret: %w", err)
	}

	if !valid {
		return nil, false, nil
	}

	return &azure.CredentialsSet{
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

func extractCredentials(secret corev1.Secret, keys extractCredentialsKeys) (*azure.Credentials, bool, error) {
	var clientJwk jose.JSONWebKey
	var err error

	clientSecret, exists := secret.Data[keys.clientSecretKey]
	passwordId, exists := secret.Data[keys.passwordIdKey]
	jwkSecret, exists := secret.Data[keys.jwkSecretKey]
	certificateId, exists := secret.Data[keys.certificateIdKey]

	if !exists {
		return nil, false, nil
	}

	err = clientJwk.UnmarshalJSON(jwkSecret)
	if err != nil {
		return nil, exists, err
	}

	return &azure.Credentials{
		Certificate: azure.Certificate{
			KeyId: string(certificateId),
			Jwk:   crypto.FromJwk(clientJwk),
		},
		Password: azure.Password{
			KeyId:        string(passwordId),
			ClientSecret: string(clientSecret),
		},
	}, exists, nil
}

func extractCurrentcredentials(secret corev1.Secret) (*azure.Credentials, bool, error) {
	keys := extractCredentialsKeys{
		certificateIdKey: CertificateIdKey,
		clientSecretKey:  ClientSecretKey,
		jwkSecretKey:     JwkKey,
		passwordIdKey:    PasswordIdKey,
	}

	return extractCredentials(secret, keys)
}

func extractNextCredentials(secret corev1.Secret) (*azure.Credentials, bool, error) {
	keys := extractCredentialsKeys{
		certificateIdKey: NextCertificateIdKey,
		clientSecretKey:  NextClientSecretKey,
		jwkSecretKey:     NextJwkKey,
		passwordIdKey:    NextPasswordIdKey,
	}

	return extractCredentials(secret, keys)
}
