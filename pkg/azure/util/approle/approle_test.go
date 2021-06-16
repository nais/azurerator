package approle_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/azure/util/approle"
	"github.com/nais/azureator/pkg/azure/util/permissions"
)

func TestNew(t *testing.T) {
	name := "role1"
	id := msgraph.UUID(uuid.New().String())

	expected := msgraph.AppRole{
		AllowedMemberTypes: []string{"Application"},
		Description:        ptr.String(name),
		DisplayName:        ptr.String(name),
		ID:                 &id,
		IsEnabled:          ptr.Bool(true),
		Value:              ptr.String(name),
	}
	actual := approle.New(id, name)

	assert.Equal(t, expected, actual)
}

func TestNewGenerateId(t *testing.T) {
	name := "role1"
	actual := approle.NewGenerateId(name)
	id := actual.ID

	expected := msgraph.AppRole{
		AllowedMemberTypes: []string{"Application"},
		Description:        ptr.String(name),
		DisplayName:        ptr.String(name),
		ID:                 id,
		IsEnabled:          ptr.Bool(true),
		Value:              ptr.String(name),
	}

	assert.Equal(t, expected, actual)
}

func TestDefaultRole(t *testing.T) {
	id := msgraph.UUID(approle.DefaultAppRoleId)
	expected := msgraph.AppRole{
		AllowedMemberTypes: []string{"Application"},
		Description:        ptr.String(approle.DefaultAppRoleValue),
		DisplayName:        ptr.String(approle.DefaultAppRoleValue),
		ID:                 &id,
		IsEnabled:          ptr.Bool(true),
		Value:              ptr.String(approle.DefaultAppRoleValue),
	}
	actual := approle.DefaultRole()

	assert.Equal(t, expected, actual)
}

func TestEnsureDefaultAppRoleIsEnabled(t *testing.T) {
	defaultRole := approle.DefaultRole()
	defaultRole.IsEnabled = ptr.Bool(false)

	roles := []msgraph.AppRole{defaultRole}
	for _, role := range roles {
		assert.False(t, *role.IsEnabled)
	}

	actual := approle.EnsureDefaultAppRoleIsEnabled(roles)
	for _, role := range actual {
		assert.True(t, *role.IsEnabled)
	}
}

func TestFromPermission(t *testing.T) {
	permission := permissions.NewGenerateIdEnabled("role")
	role := approle.FromPermission(permission)

	assert.Equal(t, "role", *role.Description)
	assert.Equal(t, "role", *role.DisplayName)
	assert.Equal(t, "role", *role.Value)
	assert.Equal(t, permission.ID, *role.ID)
}

func TestRemoveDisabled(t *testing.T) {
	enabledRole := approle.NewGenerateId("enabled-role")
	enabledRole2 := approle.NewGenerateId("enabled-role-2")
	disabledRole := approle.NewGenerateId("disabled-role")
	disabledRole.IsEnabled = ptr.Bool(false)

	roles := []msgraph.AppRole{
		enabledRole,
		enabledRole2,
		disabledRole,
	}
	application := util.EmptyApplication().
		AppRoles(roles).
		Build()

	desired := approle.RemoveDisabled(*application)
	assert.Len(t, desired, 2)
	assert.Contains(t, desired, enabledRole)
	assert.Contains(t, desired, enabledRole2)
	assert.NotContains(t, desired, disabledRole)
}
