package approleassignment

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
)

func TestDifference(t *testing.T) {
	t.Run("Same elements in both sets should return empty", func(t *testing.T) {
		a := []msgraphbeta.AppRoleAssignment{randomAppRoleAssignment()}
		b := a
		diff := Difference(a, b)
		assert.Empty(t, diff)
	})

	t.Run("Empty sets should return empty", func(t *testing.T) {
		a := make([]msgraphbeta.AppRoleAssignment, 0)
		b := make([]msgraphbeta.AppRoleAssignment, 0)
		diff := Difference(a, b)
		assert.Empty(t, diff)
	})

	t.Run("Disjoint sets should return all elements in A", func(t *testing.T) {
		a := []msgraphbeta.AppRoleAssignment{randomAppRoleAssignment()}
		b := []msgraphbeta.AppRoleAssignment{randomAppRoleAssignment()}
		diff := Difference(a, b)
		assert.NotEmpty(t, diff)
		assert.ElementsMatch(t, diff, a)
	})

	t.Run("Elements in A not in B should return relative complement of A in B", func(t *testing.T) {
		common := appRoleAssignment("test", "test", "test")
		common2 := randomAppRoleAssignment()
		revoked := []msgraphbeta.AppRoleAssignment{randomAppRoleAssignment(), randomAppRoleAssignment()}
		a := append(revoked, common, common2)
		b := []msgraphbeta.AppRoleAssignment{common, common2, randomAppRoleAssignment()}
		diff := Difference(a, b)
		assert.NotEmpty(t, diff)
		assert.NotContains(t, diff, common)
		assert.NotContains(t, diff, common2)
		assert.ElementsMatch(t, diff, revoked)
	})
}

func appRoleAssignment(appRoleId string, principalId string, resourceId string) msgraphbeta.AppRoleAssignment {
	return msgraphbeta.AppRoleAssignment{
		AppRoleID:   (*msgraphbeta.UUID)(&appRoleId),
		PrincipalID: (*msgraphbeta.UUID)(&principalId),
		ResourceID:  (*msgraphbeta.UUID)(&resourceId),
	}
}

func randomAppRoleAssignment() msgraphbeta.AppRoleAssignment {
	return appRoleAssignment(uuid.New().String(), uuid.New().String(), uuid.New().String())
}
