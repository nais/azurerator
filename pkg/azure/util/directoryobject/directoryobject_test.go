package directoryobject

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

func TestDifference(t *testing.T) {
	t.Run("Same elements in both sets should return empty", func(t *testing.T) {
		a := []msgraph.DirectoryObject{randomDirectoryObject()}
		b := a
		diff := Difference(a, b)
		assert.Empty(t, diff)
	})

	t.Run("Empty sets should return empty", func(t *testing.T) {
		a := make([]msgraph.DirectoryObject, 0)
		b := make([]msgraph.DirectoryObject, 0)
		diff := Difference(a, b)
		assert.Empty(t, diff)
	})

	t.Run("Disjoint sets should return all elements in A", func(t *testing.T) {
		a := []msgraph.DirectoryObject{randomDirectoryObject()}
		b := []msgraph.DirectoryObject{randomDirectoryObject()}
		diff := Difference(a, b)
		assert.NotEmpty(t, diff)
		assert.ElementsMatch(t, diff, a)
	})

	t.Run("Elements in A not in B should return relative complement of A in B", func(t *testing.T) {
		common := directoryObject("test")
		common2 := randomDirectoryObject()
		revoked := []msgraph.DirectoryObject{randomDirectoryObject(), randomDirectoryObject()}
		a := append(revoked, common, common2)
		b := []msgraph.DirectoryObject{common, common2, randomDirectoryObject()}
		diff := Difference(a, b)
		assert.NotEmpty(t, diff)
		assert.NotContains(t, diff, common)
		assert.NotContains(t, diff, common2)
		assert.ElementsMatch(t, diff, revoked)
	})
}

func directoryObject(id string) msgraph.DirectoryObject {
	return msgraph.DirectoryObject{
		Entity: msgraph.Entity{
			ID: &id,
		},
	}
}

func randomDirectoryObject() msgraph.DirectoryObject {
	return directoryObject(uuid.New().String())
}
