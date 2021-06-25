package approleassignment_test

import (
	"testing"

	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/client/application/approle"
	"github.com/nais/azureator/pkg/azure/client/approleassignment"
	"github.com/nais/azureator/pkg/azure/permissions"
	"github.com/nais/azureator/pkg/azure/resource"
)

const (
	target = "target-app-id"
)

var (
	role      = permissions.NewGenerateIdEnabled("some-permission")
	assignee1 = resource.Resource{
		Name:          "assignee-1",
		ObjectId:      "assignee-1-id",
		PrincipalType: resource.PrincipalTypeServicePrincipal,
	}
	assignee1AppRoleAssignment = msgraph.AppRoleAssignment{
		AppRoleID:            &role.ID,
		PrincipalDisplayName: ptr.String("assignee-1"),
		PrincipalID:          (*msgraph.UUID)(ptr.String("assignee-1-id")),
		PrincipalType:        ptr.String(string(resource.PrincipalTypeServicePrincipal)),
		ResourceID:           (*msgraph.UUID)(ptr.String(target)),
	}
	assignee2 = resource.Resource{
		Name:          "assignee-2",
		ObjectId:      "assignee-2-id",
		PrincipalType: resource.PrincipalTypeGroup,
	}
	assignee2AppRoleAssignment = msgraph.AppRoleAssignment{
		AppRoleID:            &role.ID,
		PrincipalDisplayName: ptr.String("assignee-2"),
		PrincipalID:          (*msgraph.UUID)(ptr.String("assignee-2-id")),
		PrincipalType:        ptr.String(string(resource.PrincipalTypeGroup)),
		ResourceID:           (*msgraph.UUID)(ptr.String(target)),
	}
)

func TestToAppRoleAssignments(t *testing.T) {
	assignments := assignments(resource.Resources{assignee1})

	expected := assignee1AppRoleAssignment

	assert.Len(t, assignments, 1)
	assert.Contains(t, assignments, expected)
	assert.Equal(t, assignments, approleassignment.List{expected})
}

func TestList_Has(t *testing.T) {
	assignments := assignments(resource.Resources{assignee1})

	t.Run("expected equal should return true", func(t *testing.T) {
		expected := assignee1AppRoleAssignment

		assert.True(t, assignments.Has(expected))
	})

	t.Run("different approle ID should return false", func(t *testing.T) {
		expected := assignee1AppRoleAssignment
		expected.AppRoleID = (*msgraph.UUID)(ptr.String("some-other-role"))

		assert.False(t, assignments.Has(expected))
	})

	t.Run("different principal ID should return false", func(t *testing.T) {
		expected := assignee1AppRoleAssignment
		expected.PrincipalID = (*msgraph.UUID)(ptr.String("some-other-target"))

		assert.False(t, assignments.Has(expected))
	})

	t.Run("different principal type should return false", func(t *testing.T) {
		expected := assignee1AppRoleAssignment
		expected.PrincipalType = ptr.String(string(resource.PrincipalTypeGroup))

		assert.False(t, assignments.Has(expected))
	})
}

func TestList_HasResource(t *testing.T) {
	assignments := assignments(resource.Resources{assignee1})

	t.Run("expected equal should return true", func(t *testing.T) {
		expected := assignee1

		assert.True(t, assignments.HasResource(expected))
	})

	t.Run("different object ID should return false", func(t *testing.T) {
		expected := assignee1
		expected.ObjectId = "some-other-id"

		assert.False(t, assignments.HasResource(expected))
	})

	t.Run("different principal type should return false", func(t *testing.T) {
		expected := assignee1
		expected.PrincipalType = resource.PrincipalTypeGroup

		assert.False(t, assignments.HasResource(expected))
	})
}

func TestList_FilterByRoleID(t *testing.T) {
	assignments := assignments(resource.Resources{assignee1})

	t.Run("matching role ID should return matching assignment", func(t *testing.T) {
		roleID := role.ID
		filtered := assignments.FilterByRoleID(roleID)

		assert.Len(t, filtered, 1)
		assert.Contains(t, filtered, assignee1AppRoleAssignment)
	})

	t.Run("non-matching role ID should return empty", func(t *testing.T) {
		roleID := msgraph.UUID("some-other-id")
		filtered := assignments.FilterByRoleID(roleID)

		assert.Empty(t, filtered)
		assert.NotContains(t, filtered, assignee1AppRoleAssignment)
	})
}

func TestList_FilterByType(t *testing.T) {
	assignments := assignments(resource.Resources{assignee1})

	t.Run("matching principal type should return matching assignment", func(t *testing.T) {
		principalType := resource.PrincipalTypeServicePrincipal
		filtered := assignments.FilterByType(principalType)

		assert.Len(t, filtered, 1)
		assert.Contains(t, filtered, assignee1AppRoleAssignment)
		assert.NotContains(t, filtered, assignee2AppRoleAssignment)
	})

	t.Run("non-matching principal type should return empty", func(t *testing.T) {
		principalType := resource.PrincipalTypeGroup
		filtered := assignments.FilterByType(principalType)

		assert.Empty(t, filtered)
		assert.NotContains(t, filtered, assignee1AppRoleAssignment)
		assert.NotContains(t, filtered, assignee2AppRoleAssignment)
	})
}

func TestList_Groups(t *testing.T) {
	assignments := assignments(resource.Resources{assignee1, assignee2})

	t.Run("should only return groups", func(t *testing.T) {
		filtered := assignments.Groups()

		assert.Len(t, filtered, 1)
		assert.Contains(t, filtered, assignee2AppRoleAssignment)
		assert.NotContains(t, filtered, assignee1AppRoleAssignment)
	})
}

func TestList_ServicePrincipals(t *testing.T) {
	assignments := assignments(resource.Resources{assignee1, assignee2})

	t.Run("should only return service principals", func(t *testing.T) {
		filtered := assignments.ServicePrincipals()

		assert.Len(t, filtered, 1)
		assert.Contains(t, filtered, assignee1AppRoleAssignment)
		assert.NotContains(t, filtered, assignee2AppRoleAssignment)
	})
}

func TestList_WithoutMatchingRole(t *testing.T) {
	assignments := assignments(resource.Resources{assignee1})

	t.Run("with matching role should return empty", func(t *testing.T) {
		validRoles := make(permissions.Permissions)
		validRoles.Add(role)

		result := assignments.WithoutMatchingRole(validRoles)

		assert.Empty(t, result)
		assert.NotContains(t, result, assignee1AppRoleAssignment)
	})

	t.Run("without matching role should return assignments", func(t *testing.T) {
		validRoles := make(permissions.Permissions)
		validRoles.Add(permissions.FromAppRole(approle.NewGenerateId("some-other-permission")))

		result := assignments.WithoutMatchingRole(validRoles)

		assert.Len(t, result, 1)
		assert.Contains(t, result, assignee1AppRoleAssignment)
	})
}

func assignments(resources resource.Resources) approleassignment.List {
	return approleassignment.ToAppRoleAssignments(resources, target, role)
}
