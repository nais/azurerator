package strings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/util/strings"
)

func TestRemoveDuplicates(t *testing.T) {
	list := []string{"some", "value", "some", "other", "value"}
	expected := []string{"some", "other", "value"}

	filtered := strings.RemoveDuplicates(list)

	assert.ElementsMatch(t, filtered, expected)
	assert.Len(t, filtered, len(expected))
}
