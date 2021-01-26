package secrets

import (
	"encoding/json"
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"testing"

	"github.com/nais/azureator/pkg/azure/fake"
	"github.com/nais/azureator/pkg/labels"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateSecretSpec(t *testing.T) {
	app := &v1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test",
		},
		Spec: v1.AzureAdApplicationSpec{
			SecretName: "test-secret",
			PreAuthorizedApplications: []v1.AzureAdPreAuthorizedApplication{
				{
					Application: "test-app-2",
					Namespace:   "test",
					Cluster:     "test-cluster",
				},
			},
		},
	}
	azureApp := fake.InternalAzureApp(*app)

	spec, err := spec(app, azureApp)
	assert.NoError(t, err, "should not error")

	stringData, err := stringData(azureApp)
	assert.NoError(t, err, "should not error")

	t.Run("Name should equal provided name in Spec", func(t *testing.T) {
		expected := app.Spec.SecretName
		actual := spec.Name
		assert.NotEmpty(t, actual)
		assert.Equal(t, expected, actual)
	})

	t.Run("Secret spec should be as expected", func(t *testing.T) {
		expected := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: objectMeta(app),
			StringData: stringData,
			Type:       corev1.SecretTypeOpaque,
		}
		assert.NotEmpty(t, spec)
		assert.Equal(t, expected, spec)

		assert.Equal(t, corev1.SecretTypeOpaque, spec.Type, "Secret Type should be Opaque")
	})

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run(fmt.Sprintf("Length of StringData should be equal to %v", len(AllKeys)), func(t *testing.T) {
			expected := len(AllKeys)
			assert.Len(t, spec.StringData, expected)
		})

		t.Run("Secret Data should contain Client Secret", func(t *testing.T) {
			expected := azureApp.Password.ClientSecret
			assert.Equal(t, expected, spec.StringData[ClientSecretKey])
		})

		t.Run("Secret Data should contain Private JWKS", func(t *testing.T) {
			expectedJwks := azureApp.Certificate.Jwk.ToPrivateJwks()

			expected, err := json.Marshal(expectedJwks)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), spec.StringData[JwksKey])

			var jwks jose.JSONWebKeySet
			err = json.Unmarshal([]byte(spec.StringData[JwksKey]), &jwks)
			assert.NoError(t, err)
			assert.Len(t, jwks.Keys, len(expectedJwks.Keys))
		})

		t.Run("Secret Data should contain Private JWK", func(t *testing.T) {
			expected, err := json.Marshal(azureApp.Certificate.Jwk.Private)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), spec.StringData[JwkKey])
		})

		t.Run("Secret Data should contain Certificate Key ID", func(t *testing.T) {
			expected := azureApp.Certificate.KeyId.Latest
			assert.Equal(t, expected, spec.StringData[CertificateIdKey])
		})

		t.Run("Secret Data should contain Password Key ID", func(t *testing.T) {
			expected := azureApp.Password.KeyId.Latest
			assert.Equal(t, expected, spec.StringData[PasswordIdKey])
		})

		t.Run("Secret Data should contain Client ID", func(t *testing.T) {
			expected := azureApp.ClientId
			assert.Equal(t, expected, spec.StringData[ClientIdKey])
		})

		t.Run("Secret Data should contain list of PreAuthorizedApps", func(t *testing.T) {
			var actual []azure.Resource
			err := json.Unmarshal([]byte(spec.StringData[PreAuthAppsKey]), &actual)
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
			assert.Equal(t, expected, spec.StringData[TenantId])
		})

		t.Run("Secret Data should contain well-known URL", func(t *testing.T) {
			expected := WellKnownUrl(azureApp.Tenant)
			assert.NoError(t, err)
			assert.Equal(t, expected, spec.StringData[WellKnownUrlKey])
		})
	})
}

func TestObjectMeta(t *testing.T) {
	name := "test-name"
	app := &v1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test",
		},
		Spec: v1.AzureAdApplicationSpec{
			SecretName: name,
		},
	}

	om := objectMeta(app)

	t.Run("Name should be set", func(t *testing.T) {
		actual := om.GetName()
		assert.NotEmpty(t, actual)
		assert.Equal(t, name, actual)
	})

	t.Run("Namespace should be set", func(t *testing.T) {
		actual := om.GetNamespace()
		assert.NotEmpty(t, actual)
		assert.Equal(t, app.GetNamespace(), actual)
	})
	t.Run("Labels should be set", func(t *testing.T) {
		actualLabels := om.GetLabels()
		expectedLabels := map[string]string{
			labels.AppLabelKey:  app.GetName(),
			labels.TypeLabelKey: labels.TypeLabelValue,
		}
		assert.NotEmpty(t, actualLabels, "Labels should not be empty")
		assert.Equal(t, expectedLabels, actualLabels, "Labels should be set")
	})
}
