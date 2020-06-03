package resourcecreator

import (
	"encoding/json"
	"fmt"

	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	azureConfig "github.com/nais/azureator/pkg/azure/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SecretCreator struct {
	DefaultCreator
}

const (
	CertificateIdKey = "AZURE_APP_CERTIFICATE_KEY_ID"
	ClientIdKey      = "AZURE_APP_CLIENT_ID"
	ClientSecretKey  = "AZURE_APP_CLIENT_SECRET"
	JwksPrivateKey   = "AZURE_APP_JWKS_PRIVATE"
	JwksPublicKey    = "AZURE_APP_JWKS_PUBLIC"
	PasswordIdKey    = "AZURE_APP_PASSWORD_KEY_ID"
	PreAuthAppsKey   = "AZURE_APP_PRE_AUTHORIZED_APPS"
	WellKnownUrlKey  = "AZURE_APP_WELL_KNOWN_URL"
)

var AllKeys = []string{
	CertificateIdKey,
	ClientIdKey,
	ClientSecretKey,
	JwksPrivateKey,
	JwksPublicKey,
	PasswordIdKey,
	PreAuthAppsKey,
}

func NewSecret(resource v1alpha1.AzureAdApplication, application azure.Application) Creator {
	return SecretCreator{
		DefaultCreator{
			Resource:    resource,
			Application: application,
		},
	}
}

func (c SecretCreator) Spec() (runtime.Object, error) {
	return &corev1.Secret{
		ObjectMeta: c.ObjectMeta(c.Name()),
	}, nil
}

func (c SecretCreator) MutateFn(object runtime.Object) (controllerutil.MutateFn, error) {
	secret := object.(*corev1.Secret)
	return func() error {
		data, err := c.toSecretData()
		if err != nil {
			return err
		}
		secret.StringData = data
		secret.Type = corev1.SecretTypeOpaque
		return nil
	}, nil
}

func (c SecretCreator) Name() string {
	return c.Resource.Spec.SecretName
}

func (c SecretCreator) toSecretData() (map[string]string, error) {
	jwkPrivateJson, err := json.Marshal(c.Application.Certificate.Jwks.Private)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private JWK: %w", err)
	}
	jwkPublicJson, err := json.Marshal(c.Application.Certificate.Jwks.Public)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public JWK: %w", err)
	}
	// TODO - more user friendly format?
	preAuthAppsJson, err := json.Marshal(c.Application.PreAuthorizedApps)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal preauthorized apps: %w", err)
	}
	return map[string]string{
		CertificateIdKey: c.Application.Certificate.KeyId.Latest,
		ClientIdKey:      c.Application.ClientId,
		ClientSecretKey:  c.Application.Password.ClientSecret,
		JwksPrivateKey:   string(jwkPrivateJson),
		JwksPublicKey:    string(jwkPublicJson),
		PasswordIdKey:    c.Application.Password.KeyId.Latest,
		PreAuthAppsKey:   string(preAuthAppsJson),
		WellKnownUrlKey:  azureConfig.WellKnownUrl(c.Application.Tenant),
	}, nil
}
