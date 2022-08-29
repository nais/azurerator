package annotations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/fixtures"
)

func TestAddToAnnotation(t *testing.T) {
	newValue := "new-value"

	t.Run("annotation exists", func(t *testing.T) {
		for _, tt := range []struct {
			name     string
			existing string
			expected string
		}{
			{
				name:     "one value",
				existing: "some-value",
				expected: "some-value,new-value",
			},
			{
				name:     "two values",
				existing: "some-value,some-other-value",
				expected: "some-value,some-other-value,new-value",
			},
			{
				name:     "three values",
				existing: "some-value,some-other-value,another-value",
				expected: "some-value,some-other-value,another-value,new-value",
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				app := fixtures.MinimalApplication()

				annotations.SetAnnotation(app, "some-key", tt.existing)
				val, ok := annotations.HasAnnotation(app, "some-key")
				assert.True(t, ok)
				assert.Equal(t, tt.existing, val)

				annotations.AddToAnnotation(app, "some-key", newValue)

				val, ok = annotations.HasAnnotation(app, "some-key")
				assert.True(t, ok)
				assert.Equal(t, tt.expected, val)
			})
		}
	})

	t.Run("no matching annotation", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		val, ok := annotations.HasAnnotation(app, "some-key")
		assert.False(t, ok)
		assert.Empty(t, val)

		annotations.AddToAnnotation(app, "some-key", newValue)

		val, ok = annotations.HasAnnotation(app, "some-key")
		assert.True(t, ok)
		assert.Equal(t, "new-value", val)
	})
}

func TestHasAnnotation(t *testing.T) {
	t.Run("annotation exists", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ann := make(map[string]string)
		ann["some-key"] = "some-value"
		app.SetAnnotations(ann)

		val, ok := annotations.HasAnnotation(app, "some-key")
		assert.True(t, ok)
		assert.Equal(t, "some-value", val)
	})

	t.Run("no matching annotation", func(t *testing.T) {
		app := fixtures.MinimalApplication()

		val, ok := annotations.HasAnnotation(app, "some-key")
		assert.False(t, ok)
		assert.Empty(t, val)
	})
}

func TestRemoveAnnotation(t *testing.T) {
	t.Run("annotation exists", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		annotations.SetAnnotation(app, "some-key", "some-value")
		val, ok := annotations.HasAnnotation(app, "some-key")
		assert.True(t, ok)
		assert.Equal(t, "some-value", val)

		annotations.RemoveAnnotation(app, "some-key")

		val, ok = annotations.HasAnnotation(app, "some-key")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("no matching annotation", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		val, ok := annotations.HasAnnotation(app, "some-key")
		assert.False(t, ok)
		assert.Empty(t, val)

		annotations.RemoveAnnotation(app, "some-key")

		val, ok = annotations.HasAnnotation(app, "some-key")
		assert.False(t, ok)
		assert.Empty(t, val)
	})
}

func TestRemoveFromAnnotation(t *testing.T) {
	t.Run("annotation exists, with 1 value", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		annotations.SetAnnotation(app, "some-key", "some-value")
		val, ok := annotations.HasAnnotation(app, "some-key")
		assert.True(t, ok)
		assert.Equal(t, "some-value", val)

		annotations.RemoveFromAnnotation(app, "some-key")

		val, ok = annotations.HasAnnotation(app, "some-key")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("annotation exists, with multiple values", func(t *testing.T) {
		for _, tt := range []struct {
			name     string
			value    string
			expected string
		}{
			{
				name:     "two values",
				value:    "some-value,some-other-value",
				expected: "some-other-value",
			},
			{
				name:     "three values",
				value:    "some-value,some-other-value,another-value",
				expected: "some-other-value,another-value",
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				app := fixtures.MinimalApplication()

				annotations.SetAnnotation(app, "some-key", tt.value)
				val, ok := annotations.HasAnnotation(app, "some-key")
				assert.True(t, ok)
				assert.Equal(t, tt.value, val)

				annotations.RemoveFromAnnotation(app, "some-key")

				val, ok = annotations.HasAnnotation(app, "some-key")
				assert.True(t, ok)
				assert.Equal(t, tt.expected, val)
			})
		}
	})

	t.Run("no matching annotation", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		val, ok := annotations.HasAnnotation(app, "some-key")
		assert.False(t, ok)
		assert.Empty(t, val)

		annotations.RemoveFromAnnotation(app, "some-key")

		val, ok = annotations.HasAnnotation(app, "some-key")
		assert.False(t, ok)
		assert.Empty(t, val)
	})
}

func TestSetAnnotation(t *testing.T) {
	app := fixtures.MinimalApplication()

	val, ok := annotations.HasAnnotation(app, "some-key")
	assert.False(t, ok)
	assert.Empty(t, val)

	annotations.SetAnnotation(app, "some-key", "some-value")

	val, ok = annotations.HasAnnotation(app, "some-key")
	assert.True(t, ok)
	assert.Equal(t, "some-value", val)
}
