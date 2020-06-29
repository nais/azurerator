package secrets

import (
	"encoding/json"
	"testing"

	v1 "github.com/nais/azureator/api/v1"
	azureConfig "github.com/nais/azureator/pkg/azure/config"
	"github.com/nais/azureator/pkg/fixtures/azure"
	"github.com/nais/azureator/pkg/labels"
	"github.com/stretchr/testify/assert"
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
		},
	}
	azureApp := azure.InternalAzureApp(*app)

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
		t.Run("Secret Data should contain Client Secret", func(t *testing.T) {
			expected := azureApp.Password.ClientSecret
			assert.Equal(t, expected, spec.StringData[ClientSecretKey])
		})

		t.Run("Secret Data should contain Private JWKS", func(t *testing.T) {
			expected, err := json.Marshal(azureApp.Certificate.Jwks.Private)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), spec.StringData[JwksKey])
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
			expected, err := json.Marshal(azureApp.PreAuthorizedApps)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), spec.StringData[PreAuthAppsKey])
		})

		t.Run("Secret Data should contain well-known URL", func(t *testing.T) {
			expected := azureConfig.WellKnownUrl(azureApp.Tenant)
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
