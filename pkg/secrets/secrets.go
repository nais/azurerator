package secrets

import (
	"encoding/json"
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/config"
)

const (
	CertificateIdKey     = "AZURE_APP_CERTIFICATE_KEY_ID"
	ClientIdKey          = "AZURE_APP_CLIENT_ID"
	ClientSecretKey      = "AZURE_APP_CLIENT_SECRET"
	JwkKey               = "AZURE_APP_JWK"
	JwksKey              = "AZURE_APP_JWKS"
	NextCertificateIdKey = "AZURE_APP_NEXT_CERTIFICATE_KEY_ID"
	NextClientSecretKey  = "AZURE_APP_NEXT_CLIENT_SECRET"
	NextJwkKey           = "AZURE_APP_NEXT_JWK"
	NextPasswordIdKey    = "AZURE_APP_NEXT_PASSWORD_KEY_ID"
	PasswordIdKey        = "AZURE_APP_PASSWORD_KEY_ID"
	PreAuthAppsKey       = "AZURE_APP_PRE_AUTHORIZED_APPS"
	TenantId             = "AZURE_APP_TENANT_ID"
	WellKnownUrlKey      = "AZURE_APP_WELL_KNOWN_URL"

	OpenIDConfigIssuerKey        = "AZURE_OPENID_CONFIG_ISSUER"
	OpenIDConfigJwksUriKey       = "AZURE_OPENID_CONFIG_JWKS_URI"
	OpenIDConfigTokenEndpointKey = "AZURE_OPENID_CONFIG_TOKEN_ENDPOINT"
)

var AllKeys = []string{
	CertificateIdKey,
	ClientIdKey,
	ClientSecretKey,
	JwksKey,
	JwkKey,
	NextCertificateIdKey,
	NextClientSecretKey,
	NextJwkKey,
	NextPasswordIdKey,
	PasswordIdKey,
	PreAuthAppsKey,
	TenantId,
	WellKnownUrlKey,
	OpenIDConfigIssuerKey,
	OpenIDConfigJwksUriKey,
	OpenIDConfigTokenEndpointKey,
}

func SecretData(app azure.ApplicationResult, set azure.CredentialsSet, azureOpenIDConfig config.AzureOpenIdConfig) (map[string]string, error) {
	jwkJson, err := json.Marshal(set.Current.Certificate.Jwk.Private)
	if err != nil {
		return nil, fmt.Errorf("marshalling private JWK: %w", err)
	}

	nextJwkJson, err := json.Marshal(set.Next.Certificate.Jwk.Private)
	if err != nil {
		return nil, fmt.Errorf("marshalling next private JWK: %w", err)
	}

	jwksJson, err := json.Marshal(set.Current.Certificate.Jwk.ToPrivateJwks())
	if err != nil {
		return nil, fmt.Errorf("marshalling private JWKS: %w", err)
	}

	preAuthAppsJson, err := json.Marshal(app.PreAuthorizedApps.Valid)
	if err != nil {
		return nil, fmt.Errorf("marshalling preauthorized apps: %w", err)
	}

	return map[string]string{
		CertificateIdKey:             set.Current.Certificate.KeyId,
		ClientIdKey:                  app.ClientId,
		ClientSecretKey:              set.Current.Password.ClientSecret,
		JwksKey:                      string(jwksJson),
		JwkKey:                       string(jwkJson),
		NextClientSecretKey:          set.Next.Password.ClientSecret,
		NextCertificateIdKey:         set.Next.Certificate.KeyId,
		NextJwkKey:                   string(nextJwkJson),
		NextPasswordIdKey:            set.Next.Password.KeyId,
		PasswordIdKey:                set.Current.Password.KeyId,
		PreAuthAppsKey:               string(preAuthAppsJson),
		TenantId:                     app.Tenant,
		WellKnownUrlKey:              azureOpenIDConfig.WellKnownEndpoint,
		OpenIDConfigIssuerKey:        azureOpenIDConfig.Issuer,
		OpenIDConfigJwksUriKey:       azureOpenIDConfig.JwksURI,
		OpenIDConfigTokenEndpointKey: azureOpenIDConfig.TokenEndpoint,
	}, nil
}
