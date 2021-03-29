package secrets

import (
	"encoding/json"
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/config"
	"strings"
)

const (
	DefaultKeyPrefix = "AZURE"

	certificateIdSuffix = "_APP_CERTIFICATE_KEY_ID"
	clientSecretSuffix  = "_APP_CLIENT_SECRET"
	jwkSuffix           = "_APP_JWK"
	jwksSuffix          = "_APP_JWKS"
	passwordIdSuffix    = "_APP_PASSWORD_KEY_ID"

	clientIdSuffix     = "_APP_CLIENT_ID"
	preAuthAppsSuffix  = "_APP_PRE_AUTHORIZED_APPS"
	tenantIdSuffix     = "_APP_TENANT_ID"
	wellKnownUrlSuffix = "_APP_WELL_KNOWN_URL"

	nextCertificateIdSuffix = "_APP_NEXT_CERTIFICATE_KEY_ID"
	nextClientSecretSuffix  = "_APP_NEXT_CLIENT_SECRET"
	nextJwkSuffix           = "_APP_NEXT_JWK"
	nextPasswordIdSuffix    = "_APP_NEXT_PASSWORD_KEY_ID"

	openIDConfigIssuerKey        = "_OPENID_CONFIG_ISSUER"
	openIDConfigJwksUriKey       = "_OPENID_CONFIG_JWKS_URI"
	openIDConfigTokenEndpointKey = "_OPENID_CONFIG_TOKEN_ENDPOINT"
)

type SecretDataKeys struct {
	ClientId           string
	CurrentCredentials CredentialKeys
	NextCredentials    CredentialKeys
	PreAuthApps        string
	TenantId           string
	WellKnownUrl       string
	OpenId             OpenIdConfigKeys
}

func NewSecretDataKeys(keyPrefix ...string) SecretDataKeys {
	var prefix string

	if len(keyPrefix) == 0 {
		prefix = DefaultKeyPrefix
	} else {
		prefix = secretPrefix(keyPrefix[0])
	}

	return SecretDataKeys{
		ClientId: prefix + clientIdSuffix,
		CurrentCredentials: CredentialKeys{
			CertificateKeyId: prefix + certificateIdSuffix,
			ClientSecret:     prefix + clientSecretSuffix,
			PasswordKeyId:    prefix + passwordIdSuffix,
			Jwks:             prefix + jwksSuffix,
			Jwk:              prefix + jwkSuffix,
		},
		NextCredentials: CredentialKeys{
			CertificateKeyId: prefix + nextCertificateIdSuffix,
			ClientSecret:     prefix + nextClientSecretSuffix,
			PasswordKeyId:    prefix + nextPasswordIdSuffix,
			Jwk:              prefix + nextJwkSuffix,
		},
		PreAuthApps:  prefix + preAuthAppsSuffix,
		TenantId:     prefix + tenantIdSuffix,
		WellKnownUrl: prefix + wellKnownUrlSuffix,
		OpenId: OpenIdConfigKeys{
			Issuer:        prefix + openIDConfigIssuerKey,
			JwksUri:       prefix + openIDConfigJwksUriKey,
			TokenEndpoint: prefix + openIDConfigTokenEndpointKey,
		},
	}
}

func (s SecretDataKeys) AllKeys() []string {
	return []string{
		s.ClientId,
		s.CurrentCredentials.CertificateKeyId,
		s.CurrentCredentials.ClientSecret,
		s.CurrentCredentials.PasswordKeyId,
		s.CurrentCredentials.Jwks,
		s.CurrentCredentials.Jwk,
		s.NextCredentials.CertificateKeyId,
		s.NextCredentials.ClientSecret,
		s.NextCredentials.PasswordKeyId,
		s.NextCredentials.Jwk,
		s.PreAuthApps,
		s.TenantId,
		s.WellKnownUrl,
		s.OpenId.Issuer,
		s.OpenId.JwksUri,
		s.OpenId.TokenEndpoint,
	}
}

func secretPrefix(prefix string) string {
	if len(prefix) > 0 {
		return strings.ToUpper(strings.TrimSuffix(prefix, "_"))
	}
	return DefaultKeyPrefix
}

type CredentialKeys struct {
	CertificateKeyId string
	ClientSecret     string
	PasswordKeyId    string
	Jwks             string
	Jwk              string
}

type OpenIdConfigKeys struct {
	Issuer        string
	JwksUri       string
	TokenEndpoint string
}

func SecretData(app azure.ApplicationResult, set azure.CredentialsSet, azureOpenIDConfig config.AzureOpenIdConfig, keys SecretDataKeys) (map[string]string, error) {
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
		keys.ClientId:                            app.ClientId,
		keys.CurrentCredentials.CertificateKeyId: set.Current.Certificate.KeyId,
		keys.CurrentCredentials.ClientSecret:     set.Current.Password.ClientSecret,
		keys.CurrentCredentials.Jwks:             string(jwksJson),
		keys.CurrentCredentials.Jwk:              string(jwkJson),
		keys.CurrentCredentials.PasswordKeyId:    set.Current.Password.KeyId,
		keys.NextCredentials.ClientSecret:        set.Next.Password.ClientSecret,
		keys.NextCredentials.CertificateKeyId:    set.Next.Certificate.KeyId,
		keys.NextCredentials.Jwk:                 string(nextJwkJson),
		keys.NextCredentials.PasswordKeyId:       set.Next.Password.KeyId,
		keys.PreAuthApps:                         string(preAuthAppsJson),
		keys.TenantId:                            app.Tenant,
		keys.WellKnownUrl:                        azureOpenIDConfig.WellKnownEndpoint,
		keys.OpenId.Issuer:                       azureOpenIDConfig.Issuer,
		keys.OpenId.JwksUri:                      azureOpenIDConfig.JwksURI,
		keys.OpenId.TokenEndpoint:                azureOpenIDConfig.TokenEndpoint,
	}, nil
}
