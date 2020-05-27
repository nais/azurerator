package resourcecreator

import (
	"encoding/json"
	"testing"

	"github.com/nais/azureator/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestSecretCreator(t *testing.T) {
	app := fixtures.MinimalK8sAzureAdApplication()
	azureApp := fixtures.InternalAzureApp(*app)
	c := SecretCreator{DefaultCreator{
		Resource:    *app,
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
			expected := c.Application.Credentials.Private.ClientSecret
			assert.Equal(t, expected, secret.StringData["clientSecret"])
		})

		t.Run("Secret Data should contain Private JWK", func(t *testing.T) {
			expected, err := json.Marshal(c.Application.Credentials.Private.Jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), secret.StringData[JwksSecretKey])
		})
	})
}
