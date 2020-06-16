package resourcecreator

import (
	"testing"

	"github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultCreator(t *testing.T) {
	app := v1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test",
		},
	}
	c := DefaultCreator{
		Resource:    app,
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
			AppLabelKey:  app.GetName(),
			TypeLabelKey: TypeLabelValue,
		}
		actual := om.GetLabels()
		assert.NotEmpty(t, actual)
		assert.Equal(t, expected, actual)
	})
}
