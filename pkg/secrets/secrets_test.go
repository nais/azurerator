package secrets

import (
	"encoding/json"
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/fake"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestSecretData(t *testing.T) {
	app := &v1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ap",
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
	azureApp := fake.AzureApplicationResult(*app)
	azureOpenIdConfig := fake.AzureOpenIdConfig()
	azureCredentialsSet := fake.AzureCredentialsSet(*app)

	stringData, err := SecretData(azureApp, azureCredentialsSet, azureOpenIdConfig)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		expectedLength := len(AllKeys)

		t.Run(fmt.Sprintf("Length of StringData should be equal to %v", expectedLength), func(t *testing.T) {
			assert.Len(t, stringData, expectedLength)
		})

		t.Run("Secret Data should contain Client Secret", func(t *testing.T) {
			expectedClientSecret := azureCredentialsSet.Current.Password.ClientSecret
			assert.Equal(t, expectedClientSecret, stringData[ClientSecretKey])

			expectedNextClientSecret := azureCredentialsSet.Next.Password.ClientSecret
			assert.Equal(t, expectedNextClientSecret, stringData[NextClientSecretKey])
		})

		t.Run("Secret Data should contain Private JWKS", func(t *testing.T) {
			expectedJwks := azureCredentialsSet.Current.Certificate.Jwk.ToPrivateJwks()

			expected, err := json.Marshal(expectedJwks)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[JwksKey])

			var jwks jose.JSONWebKeySet
			err = json.Unmarshal([]byte(stringData[JwksKey]), &jwks)
			assert.NoError(t, err)
			assert.Len(t, jwks.Keys, len(expectedJwks.Keys))
		})

		t.Run("Secret Data should contain Private JWK", func(t *testing.T) {
			expectedJwk, err := json.Marshal(azureCredentialsSet.Current.Certificate.Jwk.Private)

			assert.NoError(t, err)
			assert.Equal(t, string(expectedJwk), stringData[JwkKey])

			expectedNextJwk, err := json.Marshal(azureCredentialsSet.Next.Certificate.Jwk.Private)

			assert.NoError(t, err)
			assert.Equal(t, string(expectedNextJwk), stringData[NextJwkKey])
		})

		t.Run("Secret Data should contain Certificate Key ID", func(t *testing.T) {
			expectedCertificateId := azureCredentialsSet.Current.Certificate.KeyId
			assert.Equal(t, expectedCertificateId, stringData[CertificateIdKey])

			expectedNextCertificateId := azureCredentialsSet.Next.Certificate.KeyId
			assert.Equal(t, expectedNextCertificateId, stringData[NextCertificateIdKey])
		})

		t.Run("Secret Data should contain Password Key ID", func(t *testing.T) {
			expectedPasswordId := azureCredentialsSet.Current.Password.KeyId
			assert.Equal(t, expectedPasswordId, stringData[PasswordIdKey])

			expectedNextPasswordId := azureCredentialsSet.Next.Password.KeyId
			assert.Equal(t, expectedNextPasswordId, stringData[NextPasswordIdKey])
		})

		t.Run("Secret Data should contain Client ID", func(t *testing.T) {
			expected := azureApp.ClientId
			assert.Equal(t, expected, stringData[ClientIdKey])
		})

		t.Run("Secret Data should contain list of PreAuthorizedApps", func(t *testing.T) {
			var actual []azure.Resource
			err := json.Unmarshal([]byte(stringData[PreAuthAppsKey]), &actual)
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
			assert.Equal(t, expected, stringData[TenantId])
		})

		t.Run("Secret Data should contain well-known URL", func(t *testing.T) {
			expected := azureOpenIdConfig.WellKnownEndpoint
			assert.NoError(t, err)
			assert.Equal(t, expected, stringData[WellKnownUrlKey])
		})

		t.Run("Secret Data should issuer from OpenID configuration", func(t *testing.T) {
			expected := azureOpenIdConfig.Issuer
			assert.NoError(t, err)
			assert.Equal(t, expected, stringData[OpenIDConfigIssuerKey])
		})

		t.Run("Secret Data should token endpoint from OpenID configuration", func(t *testing.T) {
			expected := azureOpenIdConfig.TokenEndpoint
			assert.NoError(t, err)
			assert.Equal(t, expected, stringData[OpenIDConfigTokenEndpointKey])
		})

		t.Run("Secret Data should JWKS URI from OpenID configuration", func(t *testing.T) {
			expected := azureOpenIdConfig.JwksURI
			assert.NoError(t, err)
			assert.Equal(t, expected, stringData[OpenIDConfigJwksUriKey])
		})
	})
}
