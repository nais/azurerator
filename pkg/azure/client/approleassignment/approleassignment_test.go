package approleassignment_test

import (
	"testing"

	"github.com/google/uuid"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/client/approleassignment"
)

var (
	assignment1 = appRoleAssignment("role-id-1", "assignee-id-1", "target-id")
	assignment2 = appRoleAssignment("role-id-1", "assignee-id-2", "target-id")

	assignment3          = appRoleAssignment("role-id-1", "assignee-id-3", "target-id")
	assignment3Duplicate = appRoleAssignment("role-id-1", "assignee-id-3", "target-id")

	assignment4 = appRoleAssignment("role-id-2", "assignee-id-3", "target-id")
)

func TestDifference(t *testing.T) {
	t.Run("Same elements in both sets should return empty", func(t *testing.T) {
		a := approleassignment.List{randomAppRoleAssignment()}
		b := a
		diff := approleassignment.Difference(a, b)
		assert.Empty(t, diff)
	})

	t.Run("Empty sets should return empty", func(t *testing.T) {
		a := make(approleassignment.List, 0)
		b := make(approleassignment.List, 0)
		diff := approleassignment.Difference(a, b)
		assert.Empty(t, diff)
	})

	t.Run("Disjoint sets should return all elements in A", func(t *testing.T) {
		a := approleassignment.List{randomAppRoleAssignment()}
		b := approleassignment.List{randomAppRoleAssignment()}
		diff := approleassignment.Difference(a, b)
		assert.NotEmpty(t, diff)
		assert.ElementsMatch(t, diff, a)
	})

	t.Run("Elements in A not in B should return relative complement of A in B", func(t *testing.T) {
		common := appRoleAssignment("test", "test", "test")
		common2 := randomAppRoleAssignment()
		revoked := approleassignment.List{randomAppRoleAssignment(), randomAppRoleAssignment()}
		a := append(revoked, common, common2)
		b := approleassignment.List{common, common2, randomAppRoleAssignment()}
		diff := approleassignment.Difference(a, b)
		assert.NotEmpty(t, diff)
		assert.NotContains(t, diff, common)
		assert.NotContains(t, diff, common2)
		assert.ElementsMatch(t, diff, revoked)
	})
}

func TestToAssign(t *testing.T) {
	t.Run("should return all assignments in desired if none matching in existing", func(t *testing.T) {
		existing := approleassignment.List{assignment1, assignment2}
		desired := approleassignment.List{assignment3, assignment4}

		toAssign := approleassignment.ToAssign(existing, desired)
		assert.Len(t, toAssign, 2)
		assert.Contains(t, toAssign, assignment3)
		assert.Contains(t, toAssign, assignment4)
		assert.NotContains(t, toAssign, assignment1)
		assert.NotContains(t, toAssign, assignment2)
	})

	t.Run("should only add assignments in desired that are not in existing", func(t *testing.T) {
		existing := approleassignment.List{assignment1, assignment2}
		desired := approleassignment.List{assignment1, assignment2, assignment3, assignment4}

		toAssign := approleassignment.ToAssign(existing, desired)
		assert.Len(t, toAssign, 2)
		assert.Contains(t, toAssign, assignment3)
		assert.Contains(t, toAssign, assignment4)
		assert.NotContains(t, toAssign, assignment1)
		assert.NotContains(t, toAssign, assignment2)
	})

	t.Run("should not be marked for assignment if desired assignment already in existing", func(t *testing.T) {
		existing := approleassignment.List{assignment3}
		desired := approleassignment.List{assignment3, assignment3Duplicate}

		toAssign := approleassignment.ToAssign(existing, desired)
		assert.Len(t, toAssign, 0)
		assert.NotContains(t, toAssign, assignment3)
		assert.NotContains(t, toAssign, assignment3Duplicate)
	})
}

func TestToRevoke(t *testing.T) {
	t.Run("should return all assignments in existing if none matching in desired", func(t *testing.T) {
		existing := approleassignment.List{assignment1, assignment2}
		desired := approleassignment.List{assignment3, assignment4}

		toAssign := approleassignment.ToRevoke(existing, desired)
		assert.Len(t, toAssign, 2)
		assert.Contains(t, toAssign, assignment1)
		assert.Contains(t, toAssign, assignment2)
		assert.NotContains(t, toAssign, assignment3)
		assert.NotContains(t, toAssign, assignment4)
	})

	t.Run("should only revoke assignments in existing that are not in desired", func(t *testing.T) {
		existing := approleassignment.List{assignment1, assignment2, assignment3, assignment4}
		desired := approleassignment.List{assignment1, assignment2}

		toAssign := approleassignment.ToRevoke(existing, desired)
		assert.Len(t, toAssign, 2)
		assert.Contains(t, toAssign, assignment3)
		assert.Contains(t, toAssign, assignment4)
		assert.NotContains(t, toAssign, assignment1)
		assert.NotContains(t, toAssign, assignment2)
	})

	t.Run("should not be marked for revocation if existing assignment already in desired", func(t *testing.T) {
		existing := approleassignment.List{assignment3}
		desired := approleassignment.List{assignment3, assignment3Duplicate}

		toAssign := approleassignment.ToRevoke(existing, desired)
		assert.Len(t, toAssign, 0)
		assert.NotContains(t, toAssign, assignment3)
		assert.NotContains(t, toAssign, assignment3Duplicate)
	})
}

func TestUnmodified(t *testing.T) {
	existing := approleassignment.List{assignment1, assignment2}
	toAssign := approleassignment.List{assignment3}
	toRevoke := approleassignment.List{assignment2}

	unmodified := approleassignment.Unmodified(existing, toAssign, toRevoke)
	assert.Len(t, unmodified, 1)
	assert.Contains(t, unmodified, assignment1)
	assert.NotContains(t, unmodified, assignment2)
	assert.NotContains(t, unmodified, assignment3)
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
