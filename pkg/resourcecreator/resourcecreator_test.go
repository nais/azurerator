package resourcecreator

import (
	"testing"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCreator(t *testing.T) {
	app := fixtures.MinimalK8sAzureAdApplication()
	c := DefaultCreator{
		Resource:    *app,
		Application: azure.Application{},
	}
	name := "test-name"
	om := c.ObjectMeta(name)

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
		expected := map[string]string{
			"app":  app.GetName(),
			"type": LabelType,
		}
		actual := om.GetLabels()
		assert.NotEmpty(t, actual)
		assert.Equal(t, expected, actual)
	})
}
