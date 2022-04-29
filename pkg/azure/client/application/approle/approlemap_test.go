package approle_test

import (
	"testing"

	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/client/application/approle"
	"github.com/nais/azureator/pkg/azure/permissions"
)

func TestToMap(t *testing.T) {
	roles := make([]msgraph.AppRole, 0)
	role1 := approle.NewGenerateId("role1")
	role2 := approle.NewGenerateId("role2")
	role3 := approle.NewGenerateId("role3")
	role3Duplicate := approle.NewGenerateId("role3")
	roles = append(roles, role1)
	roles = append(roles, role2)
	roles = append(roles, role3)
	roles = append(roles, role3Duplicate)

	appRoleMap := approle.ToMap(roles)

	assert.Len(t, appRoleMap, 3)
	assert.Equal(t, role1, appRoleMap["role1"])
	assert.Equal(t, role2, appRoleMap["role2"])
	assert.Equal(t, role3, appRoleMap["role3"])
}

func TestMap_Add(t *testing.T) {
	role1 := approle.NewGenerateId("role1")
	role2 := approle.NewGenerateId("role2")
	role3 := approle.NewGenerateId("role3")
	role4 := approle.NewGenerateId("role4")

	appRoleMap := make(approle.Map)
	appRoleMap.Add(role1)
	appRoleMap.Add(role2)
	appRoleMap.Add(role3)
	appRoleMap.Add(role4)

	// duplicate roles should not be added
	role1Duplicate := approle.NewGenerateId("role1")
	appRoleMap.Add(role1Duplicate)

	assert.Len(t, appRoleMap, 4)
	assert.Equal(t, role1, appRoleMap["role1"])
	assert.Equal(t, role2, appRoleMap["role2"])
	assert.Equal(t, role3, appRoleMap["role3"])
	assert.Equal(t, role4, appRoleMap["role4"])
}

func TestMap_ToSlice(t *testing.T) {
	role1 := approle.NewGenerateId("role1")
	role2 := approle.NewGenerateId("role2")
	role3 := approle.NewGenerateId("role3")

	appRoleMap := make(approle.Map)

	appRoleMap.Add(role1)
	appRoleMap.Add(role2)
	appRoleMap.Add(role3)

	slice := appRoleMap.ToSlice()
	assert.Contains(t, slice, role1)
	assert.Contains(t, slice, role2)
	assert.Contains(t, slice, role3)
	assert.Len(t, slice, 3)
}

func TestMap_ToCreate(t *testing.T) {
	t.Run("with existing roles", func(t *testing.T) {
		existing := make(approle.Map)
		existing.Add(approle.NewGenerateId("existing-role-1"))
		existing.Add(approle.NewGenerateId("existing-role-2"))
		existing.Add(approle.DefaultRole())

		desired := make(permissions.Permissions)
		desired.Add(permissions.NewGenerateIdEnabled("existing-role-1"))
		desired.Add(permissions.NewGenerateIdEnabled("role-2"))
		desired.Add(permissions.NewGenerateIdEnabled("role-3"))

		toCreate := existing.ToCreate(desired)

		assert.Len(t, toCreate, 2)
		// should contain new roles to be created
		assert.Equal(t, approle.FromPermission(desired["role-2"]), toCreate["role-2"])
		assert.Equal(t, approle.FromPermission(desired["role-3"]), toCreate["role-3"])
		// should not contain default role
		assert.Empty(t, toCreate[permissions.DefaultAppRoleValue])
	})

	t.Run("without existing roles should add default role", func(t *testing.T) {
		existing := make(approle.Map)

		desired := make(permissions.Permissions)
		desired.Add(permissions.NewGenerateIdEnabled("role-1"))

		toCreate := existing.ToCreate(desired)
		assert.Len(t, toCreate, 2)
		// should contain the new roles to be created
		assert.Equal(t, approle.FromPermission(desired["role-1"]), toCreate["role-1"])
		// should contain default role if not in existing
		assert.Equal(t, approle.DefaultRole(), toCreate[permissions.DefaultAppRoleValue])
	})
}

func TestMap_ToDisable(t *testing.T) {
	existing := make(approle.Map)
	existing.Add(approle.NewGenerateId("existing-role-1"))
	existing.Add(approle.NewGenerateId("existing-role-2"))

	desired := make(permissions.Permissions)
	desired.Add(permissions.NewGenerateIdEnabled("existing-role-1"))
	desired.Add(permissions.NewGenerateIdEnabled("role-2"))

	toDisable := existing.ToDisable(desired)

	assert.Len(t, toDisable, 1)

	// should contain non-desired roles which should be disabled
	nonDesired := existing["existing-role-2"]
	nonDesired.IsEnabled = ptr.Bool(false)

	assert.Equal(t, nonDesired, toDisable["existing-role-2"])

	// should not disable default role
	assert.NotContains(t, toDisable, approle.DefaultRole())
}

func TestMap_Unmodified(t *testing.T) {
	existing := make(approle.Map)
	existingRole1 := approle.NewGenerateId("existing-role-1")
	existingRole1.AllowedMemberTypes = []string{"Application", "User"}
	existingRole1.Description = ptr.String("non standard description")
	existingRole1.DisplayName = ptr.String("non standard display name")
	existingRole1.IsEnabled = ptr.Bool(false)
	existingRole1.Origin = ptr.String("some origin")
	existing.Add(existingRole1)
	existing.Add(approle.NewGenerateId("existing-role-2"))

	desired := make(permissions.Permissions)
	desired.Add(permissions.NewGenerateIdEnabled("existing-role-1"))
	desired.Add(permissions.NewGenerateIdEnabled("role-2"))

	toCreate := existing.ToCreate(desired)
	toDisable := existing.ToDisable(desired)

	unmodified := existing.Unmodified(toCreate, toDisable)

	assert.Len(t, unmodified, 1)
	// should contain non-modified roles
	assert.Equal(t, *existing["existing-role-1"].ID, *unmodified["existing-role-1"].ID)
	assert.Equal(t, "existing-role-1", *unmodified["existing-role-1"].Value)

	// unmodified roles should conform to standard
	assert.Equal(t, "existing-role-1", *unmodified["existing-role-1"].Description)
	assert.Equal(t, "existing-role-1", *unmodified["existing-role-1"].DisplayName)
	assert.Nil(t, unmodified["existing-role-1"].Origin)
	assert.True(t, *unmodified["existing-role-1"].IsEnabled)
	assert.Len(t, unmodified["existing-role-1"].AllowedMemberTypes, 1)
	assert.Contains(t, unmodified["existing-role-1"].AllowedMemberTypes, "Application")
}
