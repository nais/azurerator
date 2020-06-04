package resourcecreator

import (
	"encoding/json"
	"testing"

	"github.com/nais/azureator/api/v1alpha1"
	azureConfig "github.com/nais/azureator/pkg/azure/config"
	"github.com/nais/azureator/pkg/fixtures/azure"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecretCreator(t *testing.T) {
	app := v1alpha1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test",
		},
		Spec: v1alpha1.AzureAdApplicationSpec{
			SecretName: "test-secret",
		},
	}
	azureApp := azure.InternalAzureApp(app)
	c := SecretCreator{DefaultCreator{
		Resource:    app,
		Application: azureApp,
	}}

	t.Run("Name should equal provided name in Spec", func(t *testing.T) {
		expected := app.Spec.SecretName
		actual := c.Name()
		assert.NotEmpty(t, actual)
		assert.Equal(t, expected, actual)
	})

	t.Run("Secret spec should be as expected", func(t *testing.T) {
		expected := &corev1.Secret{
			ObjectMeta: c.ObjectMeta(c.Name()),
		}
		actual, err := c.Spec()
		assert.NoError(t, err)
		assert.NotEmpty(t, actual)
		assert.Equal(t, expected, actual)
	})

	t.Run("MutateFn should perform expected mutations", func(t *testing.T) {
		spec, err := c.Spec()
		assert.NoError(t, err)

		fn, err := c.MutateFn(spec)
		assert.NoError(t, err)
		assert.NoError(t, fn())

		secret := spec.(*corev1.Secret)
		t.Run("Secret Type should be Opaque", func(t *testing.T) {
			expected := corev1.SecretTypeOpaque
			assert.Equal(t, expected, secret.Type)
		})

		t.Run("Secret Data should contain Client Secret", func(t *testing.T) {
			expected := c.Application.Password.ClientSecret
			assert.Equal(t, expected, secret.StringData[ClientSecretKey])
		})

		t.Run("Secret Data should contain Private JWKS", func(t *testing.T) {
			expected, err := json.Marshal(c.Application.Certificate.Jwks.Private)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), secret.StringData[JwksKey])
		})

		t.Run("Secret Data should contain Certificate Key ID", func(t *testing.T) {
			expected := c.Application.Certificate.KeyId.Latest
			assert.Equal(t, expected, secret.StringData[CertificateIdKey])
		})

		t.Run("Secret Data should contain Password Key ID", func(t *testing.T) {
			expected := c.Application.Password.KeyId.Latest
			assert.Equal(t, expected, secret.StringData[PasswordIdKey])
		})

		t.Run("Secret Data should contain Client ID", func(t *testing.T) {
			expected := c.Application.ClientId
			assert.Equal(t, expected, secret.StringData[ClientIdKey])
		})

		t.Run("Secret Data should contain list of PreAuthorizedApps", func(t *testing.T) {
			expected, err := json.Marshal(c.Application.PreAuthorizedApps)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), secret.StringData[PreAuthAppsKey])
		})

		t.Run("Secret Data should contain well-known URL", func(t *testing.T) {
			expected := azureConfig.WellKnownUrl(c.Application.Tenant)
			assert.NoError(t, err)
			assert.Equal(t, expected, secret.StringData[WellKnownUrlKey])
		})
	})
}
