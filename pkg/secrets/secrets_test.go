package secrets

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/fake"
)

const (
	AllSecretKeyCount = 16
)

func TestSecretData(t *testing.T) {
	app := &v1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test",
		},
		Spec: v1.AzureAdApplicationSpec{
			SecretName: "test-secret",
			PreAuthorizedApplications: []v1.AccessPolicyRule{
				{
					Application: "test-app-2",
					Namespace:   "test",
					Cluster:     "test-cluster",
				},
			},
		},
	}
	azureApp := fake.AzureApplicationResult(*app, azure.OperationResultCreated)
	azureOpenIdConfig := fake.AzureOpenIdConfig()
	azureCredentialsSet := fake.AzureCredentialsSet(*app)

	keys := NewSecretDataKeys()
	stringData, err := SecretData(azureApp, azureCredentialsSet, azureOpenIdConfig, keys)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run(fmt.Sprintf("Length of StringData should be equal to %v", AllSecretKeyCount), func(t *testing.T) {
			assert.Len(t, stringData, AllSecretKeyCount)
		})

		t.Run("Secret Data should contain Client Secret", func(t *testing.T) {
			expectedClientSecret := azureCredentialsSet.Current.Password.ClientSecret
			assert.Equal(t, expectedClientSecret, stringData[keys.CurrentCredentials.ClientSecret])

			expectedNextClientSecret := azureCredentialsSet.Next.Password.ClientSecret
			assert.Equal(t, expectedNextClientSecret, stringData[keys.NextCredentials.ClientSecret])
		})

		t.Run("Secret Data should contain Private JWKS", func(t *testing.T) {
			expectedJwks := azureCredentialsSet.Current.Certificate.Jwk.ToPrivateJwks()

			expected, err := json.Marshal(expectedJwks)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[keys.CurrentCredentials.Jwks])

			var jwks jose.JSONWebKeySet
			err = json.Unmarshal([]byte(stringData[keys.CurrentCredentials.Jwks]), &jwks)
			assert.NoError(t, err)
			assert.Len(t, jwks.Keys, len(expectedJwks.Keys))
		})

		t.Run("Secret Data should contain Private JWK", func(t *testing.T) {
			expectedJwk, err := json.Marshal(azureCredentialsSet.Current.Certificate.Jwk.Private)

			assert.NoError(t, err)
			assert.Equal(t, string(expectedJwk), stringData[keys.CurrentCredentials.Jwk])

			expectedNextJwk, err := json.Marshal(azureCredentialsSet.Next.Certificate.Jwk.Private)

			assert.NoError(t, err)
			assert.Equal(t, string(expectedNextJwk), stringData[keys.NextCredentials.Jwk])
		})

		t.Run("Secret Data should contain Certificate Key ID", func(t *testing.T) {
			expectedCertificateId := azureCredentialsSet.Current.Certificate.KeyId
			assert.Equal(t, expectedCertificateId, stringData[keys.CurrentCredentials.CertificateKeyId])

			expectedNextCertificateId := azureCredentialsSet.Next.Certificate.KeyId
			assert.Equal(t, expectedNextCertificateId, stringData[keys.NextCredentials.CertificateKeyId])
		})

		t.Run("Secret Data should contain Password Key ID", func(t *testing.T) {
			expectedPasswordId := azureCredentialsSet.Current.Password.KeyId
			assert.Equal(t, expectedPasswordId, stringData[keys.CurrentCredentials.PasswordKeyId])

			expectedNextPasswordId := azureCredentialsSet.Next.Password.KeyId
			assert.Equal(t, expectedNextPasswordId, stringData[keys.NextCredentials.PasswordKeyId])
		})

		t.Run("Secret Data should contain Client ID", func(t *testing.T) {
			expected := azureApp.ClientId
			assert.Equal(t, expected, stringData[keys.ClientId])
		})

		t.Run("Secret Data should contain list of PreAuthorizedApps", func(t *testing.T) {
			var actual []azure.Resource
			err := json.Unmarshal([]byte(stringData[keys.PreAuthApps]), &actual)
			assert.NoError(t, err)
			assert.Len(t, actual, 1)
			assert.Empty(t, actual[0].PrincipalType)
			assert.Empty(t, actual[0].ObjectId)
			assert.NotEmpty(t, actual[0].ClientId)
			assert.Equal(t, "test-cluster:test:test-app-2", actual[0].Name)
		})

		t.Run("Secret Data should contain tenant ID", func(t *testing.T) {
			expected := azureApp.Tenant
			assert.NoError(t, err)
			assert.Equal(t, expected, stringData[keys.TenantId])
		})

		t.Run("Secret Data should contain well-known URL", func(t *testing.T) {
			expected := azureOpenIdConfig.WellKnownEndpoint
			assert.NoError(t, err)
			assert.Equal(t, expected, stringData[keys.WellKnownUrl])
		})

		t.Run("Secret Data should issuer from OpenID configuration", func(t *testing.T) {
			expected := azureOpenIdConfig.Issuer
			assert.NoError(t, err)
			assert.Equal(t, expected, stringData[keys.OpenId.Issuer])
		})

		t.Run("Secret Data should token endpoint from OpenID configuration", func(t *testing.T) {
			expected := azureOpenIdConfig.TokenEndpoint
			assert.NoError(t, err)
			assert.Equal(t, expected, stringData[keys.OpenId.TokenEndpoint])
		})

		t.Run("Secret Data should JWKS URI from OpenID configuration", func(t *testing.T) {
			expected := azureOpenIdConfig.JwksURI
			assert.NoError(t, err)
			assert.Equal(t, expected, stringData[keys.OpenId.JwksUri])
		})
	})
}

func TestNewSecretDataKeys(t *testing.T) {
	t.Run("SecretDataKeys with no args should return keys with default prefix", func(t *testing.T) {
		keys := NewSecretDataKeys()
		assertAllKeysWithPrefix(t, keys, "AZURE")
	})

	t.Run("SecretDataKeys with single args should return keys with set prefix", func(t *testing.T) {
		prefix := "the-best-prefix"
		keys := NewSecretDataKeys(prefix)
		assertAllKeysWithPrefix(t, keys, "THE-BEST-PREFIX")
	})

	t.Run("SecretDataKeys with single empty string args should return keys with default prefix", func(t *testing.T) {
		keys := NewSecretDataKeys("")
		assertAllKeysWithPrefix(t, keys, "AZURE")
	})

	t.Run("SecretDataKeys with multiple args should return keys with first arg as prefix", func(t *testing.T) {
		prefix := "some-prefix"
		secondPrefix := "some-other-prefix"

		keys := NewSecretDataKeys(prefix, secondPrefix)
		assertAllKeysWithPrefix(t, keys, "SOME-PREFIX")
	})

	t.Run("SecretDataKeys with unnecessary suffix should be stripped", func(t *testing.T) {
		prefix := "PREFIX_"

		keys := NewSecretDataKeys(prefix)
		assertAllKeysWithPrefix(t, keys, "PREFIX")
	})
}

func assertAllKeysWithPrefix(t *testing.T, keys SecretDataKeys, prefix string) {
	allKeys := keys.AllKeys()

	assert.Len(t, allKeys, AllSecretKeyCount)

	for _, key := range allKeys {
		assert.True(t, strings.HasPrefix(key, prefix))
	}
}
