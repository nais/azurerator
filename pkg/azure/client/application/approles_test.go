package application_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/client/application"
	"github.com/nais/azureator/pkg/azure/util/approle"
	"github.com/nais/azureator/pkg/azure/util/permissions"
)

func TestAppRoles_DescribeCreate_DesiredIsEmpty_ShouldAddDefaultRole(t *testing.T) {
	desired := make(permissions.Permissions)
	roles := application.Application{}.AppRoles().DescribeCreate(desired)

	assert.Len(t, roles, 1)
	assertContainsDefaultRole(t, roles)
}

func TestAppRoles_DescribeCreate_DisabledDefaultRoleInDesired_ShouldNotDisable(t *testing.T) {
	desired := make(permissions.Permissions)
	defaultRole := approle.DefaultRole()
	defaultRole.IsEnabled = ptr.Bool(false)
	desired.Add(permissions.FromAppRole(defaultRole))

	roles := application.Application{}.AppRoles().DescribeCreate(desired)

	assert.Len(t, roles, 1)
	assertContainsDefaultRole(t, roles)
}

func TestAppRoles_DescribeCreate_CustomRoles_ShouldAddCustomRolesAndDefaultRole(t *testing.T) {
	role1 := approle.NewGenerateId("role-1")
	role2 := approle.NewGenerateId("role-2")

	desired := make(permissions.Permissions)
	desired.Add(permissions.FromAppRole(role1))
	desired.Add(permissions.FromAppRole(role2))

	roles := application.Application{}.AppRoles().DescribeCreate(desired)

	assertContainsRole(t, role1, roles)
	assertContainsRole(t, role2, roles)
	assertContainsDefaultRole(t, roles)

	// assert that length of AppRoles equals length of the resulting union set of (desired + Default AppRole)
	assert.Len(t, roles, len(desired)+1)
}

func TestAppRoles_DescribeUpdate_DefaultRoleNotExistsInDesiredNorExisting_ShouldAdd(t *testing.T) {
	desired := make(permissions.Permissions)
	existing := make([]msgraph.AppRole, 0)
	roles := application.Application{}.AppRoles().DescribeUpdate(desired, existing)

	assert.Len(t, roles, 1)
	assertContainsDefaultRole(t, roles)
}

func TestAppRoles_DescribeUpdate_DefaultRoleAlreadyExists_ShouldOnlyEnable(t *testing.T) {
	desired := make(permissions.Permissions)
	existing := []msgraph.AppRole{
		approle.DefaultRole(),
	}
	roles := application.Application{}.AppRoles().DescribeUpdate(desired, existing)

	assert.Len(t, roles, 1)
	assertContainsDefaultRole(t, roles)

	// test case where the default role was previously created by another entity with differing values in the fields
	id := msgraph.UUID(uuid.New().String())
	existing = []msgraph.AppRole{
		{
			AllowedMemberTypes: []string{"Application"},
			Description:        ptr.String("Description"),
			DisplayName:        ptr.String("DisplayName"),
			ID:                 &id,
			IsEnabled:          ptr.Bool(false),
			Value:              ptr.String(approle.DefaultAppRoleValue),
		},
	}
	roles = application.Application{}.AppRoles().DescribeUpdate(desired, existing)

	assert.Len(t, roles, 1)
	assertContainsDefaultRoleWithLambda(t, roles, func(t assert.TestingT, expected, actual msgraph.AppRole) {
		assert.Len(t, actual.AllowedMemberTypes, 1)
		assert.Contains(t, actual.AllowedMemberTypes, "Application")
		assert.Equal(t, *actual.Description, approle.DefaultAppRoleValue)
		assert.Equal(t, *actual.DisplayName, approle.DefaultAppRoleValue)
		assert.Equal(t, *actual.ID, id)
		assert.True(t, *actual.IsEnabled)
		assert.Equal(t, *actual.Value, approle.DefaultAppRoleValue)
	})
}

func TestAppRoles_DescribeUpdate_DisabledDefaultRoleInDesired_ShouldNotDisable(t *testing.T) {
	desired := make(permissions.Permissions)
	desired.Add(permissions.NewGenerateIdDisabled(approle.DefaultAppRoleValue))

	existing := make([]msgraph.AppRole, 0)
	roles := application.Application{}.AppRoles().DescribeUpdate(desired, existing)

	assert.Len(t, roles, 1)
	assertContainsDefaultRole(t, roles)
}

func TestAppRoles_DescribeUpdate_CustomRoles_ShouldAddDesiredAndRemoveNonDesiredRoles(t *testing.T) {
	role1 := approle.NewGenerateId("role-1")
	role2 := approle.NewGenerateId("role-2")
	role3 := approle.NewGenerateId("role-3")

	desired := make(permissions.Permissions)
	desired.Add(permissions.NewGenerateIdDisabled(approle.DefaultAppRoleValue))
	desired.Add(permissions.FromAppRole(role1))
	desired.Add(permissions.FromAppRole(role2))

	existing := []msgraph.AppRole{
		approle.DefaultRole(),
		role1,
		role3,
	}

	roles := application.Application{}.AppRoles().DescribeUpdate(desired, existing)

	// assert that role "role-1" still exists and is unmodified
	assertContainsRole(t, role1, roles)
	// assert that role "role-2" is added
	assertContainsRole(t, role2, roles)
	// assert that the default role still exists and is unmodified
	assertContainsDefaultRole(t, roles)

	// assert that the non-desired role "role-3" still exists and is set to disabled
	assertContainsRoleWithLambda(t, role3, roles, func(t assert.TestingT, expected, actual msgraph.AppRole) {
		assert.Equal(t, *expected.Value, *actual.Value)
		assert.Equal(t, *expected.ID, *actual.ID)
		assert.Equal(t, *expected.Value, *actual.DisplayName)
		assert.Equal(t, *expected.Value, *actual.Description)
		assert.False(t, *actual.IsEnabled)
		assert.Contains(t, actual.AllowedMemberTypes, "Application")
		assert.Len(t, actual.AllowedMemberTypes, 1)
	})

	// assert that length of AppRoles equals length of the resulting union set of (existing + desired)
	assert.Len(t, roles, len(desired)+1)
}

func defaultRoleAsserter() func(t assert.TestingT, expected, actual msgraph.AppRole) {
	return func(t assert.TestingT, expected, actual msgraph.AppRole) {
		assert.Equal(t, *expected.Value, *actual.Value)
		assert.Equal(t, *expected.ID, *actual.ID)
		assert.Equal(t, *expected.Value, *actual.DisplayName)
		assert.Equal(t, *expected.Value, *actual.Description)
		assert.True(t, *actual.IsEnabled)
		assert.Contains(t, actual.AllowedMemberTypes, "Application")
		assert.Len(t, actual.AllowedMemberTypes, 1)
	}
}

func assertContainsRole(t assert.TestingT, expected msgraph.AppRole, roles []msgraph.AppRole) {
	assertContainsRoleWithLambda(t, expected, roles, defaultRoleAsserter())
}

func assertContainsRoleWithLambda(t assert.TestingT, expected msgraph.AppRole, roles []msgraph.AppRole, validatingFunc func(t assert.TestingT, expected, actual msgraph.AppRole)) {
	found := false
	for _, role := range roles {
		if *role.Value == *expected.Value {
			found = true
			validatingFunc(t, expected, role)
			break
		}
	}
	assert.True(t, found)
}

func assertContainsDefaultRole(t assert.TestingT, roles []msgraph.AppRole) {
	assertContainsDefaultRoleWithLambda(t, roles, defaultRoleAsserter())
}

func assertContainsDefaultRoleWithLambda(t assert.TestingT, roles []msgraph.AppRole, validatingFunc func(t assert.TestingT, expected, actual msgraph.AppRole)) {
	assertContainsRoleWithLambda(t, approle.DefaultRole(), roles, validatingFunc)
}
