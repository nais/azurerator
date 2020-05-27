package resourcecreator

import (
	"encoding/json"
	"testing"

	"github.com/nais/azureator/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestConfigMapCreator(t *testing.T) {
	app := fixtures.MinimalK8sAzureAdApplication()
	azureApp := fixtures.InternalAzureApp(*app)
	c := ConfigMapCreator{DefaultCreator{
		Resource:    *app,
		Application: azureApp,
	}}

	t.Run("Name should equal provided name in Spec", func(t *testing.T) {
		expected := app.Spec.ConfigMapName
		actual := c.Name()
		assert.NotEmpty(t, actual)
		assert.Equal(t, expected, actual)
	})

	t.Run("ConfigMap spec should be as expected", func(t *testing.T) {
		expected := &corev1.ConfigMap{
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

		configMap := spec.(*corev1.ConfigMap)

		t.Run("ConfigMap Data should contain Client ID", func(t *testing.T) {
			expected := c.Application.Credentials.Public.ClientId
			assert.Equal(t, expected, configMap.Data["clientId"])
		})

		t.Run("ConfigMap Data should contain list of PreAuthorizedApps", func(t *testing.T) {
			expected, err := json.Marshal(c.Application.PreAuthorizedApps)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), configMap.Data["preAuthorizedApps"])
		})

		t.Run("ConfigMap Data should contain Public JWK", func(t *testing.T) {
			expected, err := json.Marshal(c.Application.Credentials.Public.Jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), configMap.Data[JwksSecretKey])
		})
	})
}
