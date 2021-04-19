package approleassignment

import (
	"testing"

	"github.com/google/uuid"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"
)

func TestDifference(t *testing.T) {
	t.Run("Same elements in both sets should return empty", func(t *testing.T) {
		a := []msgraph.AppRoleAssignment{randomAppRoleAssignment()}
		b := a
		diff := Difference(a, b)
		assert.Empty(t, diff)
	})

	t.Run("Empty sets should return empty", func(t *testing.T) {
		a := make([]msgraph.AppRoleAssignment, 0)
		b := make([]msgraph.AppRoleAssignment, 0)
		diff := Difference(a, b)
		assert.Empty(t, diff)
	})

	t.Run("Disjoint sets should return all elements in A", func(t *testing.T) {
		a := []msgraph.AppRoleAssignment{randomAppRoleAssignment()}
		b := []msgraph.AppRoleAssignment{randomAppRoleAssignment()}
		diff := Difference(a, b)
		assert.NotEmpty(t, diff)
		assert.ElementsMatch(t, diff, a)
	})

	t.Run("Elements in A not in B should return relative complement of A in B", func(t *testing.T) {
		common := appRoleAssignment("test", "test", "test")
		common2 := randomAppRoleAssignment()
		revoked := []msgraph.AppRoleAssignment{randomAppRoleAssignment(), randomAppRoleAssignment()}
		a := append(revoked, common, common2)
		b := []msgraph.AppRoleAssignment{common, common2, randomAppRoleAssignment()}
		diff := Difference(a, b)
		assert.NotEmpty(t, diff)
		assert.NotContains(t, diff, common)
		assert.NotContains(t, diff, common2)
		assert.ElementsMatch(t, diff, revoked)
	})
}

func appRoleAssignment(appRoleId string, principalId string, resourceId string) msgraph.AppRoleAssignment {
	return msgraph.AppRoleAssignment{
		AppRoleID:   (*msgraph.UUID)(&appRoleId),
		PrincipalID: (*msgraph.UUID)(&principalId),
		ResourceID:  (*msgraph.UUID)(&resourceId),
	}
}

func randomAppRoleAssignment() msgraph.AppRoleAssignment {
	return appRoleAssignment(uuid.New().String(), uuid.New().String(), uuid.New().String())
}
