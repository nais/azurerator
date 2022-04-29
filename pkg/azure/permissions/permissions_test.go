package permissions_test

import (
	"fmt"
	"testing"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/azureator/pkg/azure/client/application/approle"
	"github.com/nais/azureator/pkg/azure/client/application/permissionscope"
	"github.com/nais/azureator/pkg/azure/permissions"
)

func TestGenerateDesiredPermissionSet(t *testing.T) {
	app := minimalApplication()
	desired := permissions.GenerateDesiredPermissionSet(*app)

	assert.Len(t, desired, 2)
	assertContainsDefaultPermissions(t, desired)

	app.Spec.PreAuthorizedApplications = []naisiov1.AccessPolicyInboundRule{
		{
			AccessPolicyRule: naisiov1.AccessPolicyRule{
				Application: "a",
			},
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles:  []naisiov1.AccessPolicyPermission{"read", "write"},
				Scopes: []naisiov1.AccessPolicyPermission{"admin"},
			},
		}, {
			AccessPolicyRule: naisiov1.AccessPolicyRule{
				Application: "b",
			},
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles:  []naisiov1.AccessPolicyPermission{"read"},
				Scopes: []naisiov1.AccessPolicyPermission{"write"},
			},
		}, {
			AccessPolicyRule: naisiov1.AccessPolicyRule{
				Application: "c",
			},
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles:  []naisiov1.AccessPolicyPermission{"write", "admin"},
				Scopes: []naisiov1.AccessPolicyPermission{"read"},
			},
		},
	}

	desired = permissions.GenerateDesiredPermissionSet(*app)
	assert.Len(t, desired, 5)
	assertPermissionsInPermissions(t, desired, []naisiov1.AccessPolicyPermission{"read", "write", "admin"})
}

func TestExtractPermissions(t *testing.T) {
	msgraphApp := minimalMsGraphApplication()
	actual := permissions.ExtractPermissions(msgraphApp)
	expected := []naisiov1.AccessPolicyPermission{
		"role-1", "role-2", "role-3",
		"scope-1", "scope-2",
		"common", "common-2",
	}
	assert.Len(t, actual, len(expected))

	assertPermissionsInPermissions(t, actual, expected)
	assertPermissionIDsMatch(t, msgraphApp, actual)
}

func TestGenerateDesiredPermissionSetPreserveExisting(t *testing.T) {
	existing := minimalMsGraphApplication()
	app := minimalApplication()
	app.Spec.PreAuthorizedApplications = []naisiov1.AccessPolicyInboundRule{
		{
			AccessPolicyRule: naisiov1.AccessPolicyRule{
				Application: "a",
			},
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles:  []naisiov1.AccessPolicyPermission{"role-1"},
				Scopes: []naisiov1.AccessPolicyPermission{"scope-1", "common"},
			},
		}, {
			AccessPolicyRule: naisiov1.AccessPolicyRule{
				Application: "b",
			},
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles:  []naisiov1.AccessPolicyPermission{"role-3", "common"},
				Scopes: []naisiov1.AccessPolicyPermission{"scope-2", "scope-3"},
			},
		},
	}

	desired := permissions.GenerateDesiredPermissionSetPreserveExisting(*app, *existing)
	expected := []naisiov1.AccessPolicyPermission{"role-1", "role-3", "scope-1", "scope-2", "scope-3", "common"}

	// length of desired set is set of roles + set of scopes in .Spec.PreAuthorizedApplications + the default permissions
	assert.Len(t, desired, len(expected)+2)

	assertPermissionsInPermissions(t, desired, expected)
	assertPermissionIDsMatch(t, existing, desired)
	assertContainsDefaultPermissions(t, desired)

	// assert that non-desired role "role-2" is removed
	assert.NotContains(t, desired, "role-2")
	// assert that non-desired scope and role "common-2" is removed
	assert.NotContains(t, desired, "common-2")
}

func TestGenerateDesiredPermissionSetPreserveExisting_LegacyApplication(t *testing.T) {
	existing := legacyMsGraphApplication()
	app := minimalApplication()

	desired := permissions.GenerateDesiredPermissionSetPreserveExisting(*app, *existing)
	expected := []naisiov1.AccessPolicyPermission{
		naisiov1.AccessPolicyPermission(permissions.DefaultPermissionScopeValue),
		naisiov1.AccessPolicyPermission(permissions.DefaultAppRoleValue),
	}

	assert.Len(t, desired, len(expected))

	assertPermissionsInPermissions(t, desired, expected)

	// existing permission IDs should be preserved in desired permission IDs
	assertPermissionIDsMatch(t, existing, desired)
}

func TestPermissions_Add(t *testing.T) {
	result := make(permissions.Permissions)
	existing := permissions.NewGenerateIdEnabled("existing")
	result["existing"] = existing

	existingDuplicate := permissions.NewGenerateIdDisabled("existing")
	result.Add(existingDuplicate)
	newPermission := permissions.NewGenerateIdEnabled("new")
	result.Add(newPermission)
	newPermission2 := permissions.NewGenerateIdEnabled("new-2")
	result.Add(newPermission2)

	assert.Len(t, result, 3)
	assert.Equal(t, existing, result["existing"])
	assert.NotEqual(t, existingDuplicate, result["existing"])
	assert.Equal(t, newPermission, result["new"])
	assert.Equal(t, newPermission2, result["new-2"])
}

func TestPermissions_Filter(t *testing.T) {
	result := make(permissions.Permissions)
	result.Add(permissions.NewGenerateIdEnabled("permission-1"))
	result.Add(permissions.NewGenerateIdEnabled("permission-2"))
	result.Add(permissions.NewGenerateIdEnabled("permission-3"))

	desired := make([]string, 0)
	desired = append(desired, "permission-1", "permission-2", "permission-non-existing")

	filtered := result.Filter(desired...)
	assert.Len(t, filtered, 2)
	assert.Equal(t, filtered["permission-1"], result["permission-1"])
	assert.Equal(t, filtered["permission-2"], result["permission-2"])
	assert.Empty(t, filtered["permission-3"])
	assert.Empty(t, filtered["permission-non-existing"])
}

func TestPermissions_PermissionIDs(t *testing.T) {
	result := make(permissions.Permissions)
	result.Add(permissions.NewGenerateIdEnabled("permission-1"))
	result.Add(permissions.NewGenerateIdEnabled("permission-2"))
	result.Add(permissions.NewGenerateIdEnabled("permission-3"))

	permissionIDs := result.PermissionIDs()

	assert.Len(t, permissionIDs, 3)
	assert.Contains(t, permissionIDs, string(result["permission-1"].ID))
	assert.Contains(t, permissionIDs, string(result["permission-2"].ID))
	assert.Contains(t, permissionIDs, string(result["permission-3"].ID))
}

func TestPermissions_Enabled(t *testing.T) {
	result := make(permissions.Permissions)
	result.Add(permissions.NewGenerateIdEnabled("permission-1"))
	result.Add(permissions.NewGenerateIdDisabled("permission-2"))
	result.Add(permissions.NewGenerateIdEnabled("permission-3"))
	result.Add(permissions.NewGenerateIdDisabled("permission-4"))

	enabled := result.Enabled()

	assert.Len(t, enabled, 2)
	assert.Equal(t, enabled["permission-1"], result["permission-1"])
	assert.Equal(t, enabled["permission-3"], result["permission-3"])
	assert.Empty(t, enabled["permission-2"])
	assert.Empty(t, enabled["permission-4"])
}

func TestPermissions_Disabled(t *testing.T) {
	result := make(permissions.Permissions)
	result.Add(permissions.NewGenerateIdEnabled("permission-1"))
	result.Add(permissions.NewGenerateIdDisabled("permission-2"))
	result.Add(permissions.NewGenerateIdEnabled("permission-3"))
	result.Add(permissions.NewGenerateIdDisabled("permission-4"))

	disabled := result.Disabled()

	assert.Len(t, disabled, 2)
	assert.Equal(t, disabled["permission-2"], result["permission-2"])
	assert.Equal(t, disabled["permission-4"], result["permission-4"])
	assert.Empty(t, disabled["permission-1"])
	assert.Empty(t, disabled["permission-3"])
}

func TestPermissions_HasRoleID(t *testing.T) {
	result := make(permissions.Permissions)
	result.Add(permissions.New("id-1", "permission-1", true))
	result.Add(permissions.New("id-2", "permission-2", true))
	result.Add(permissions.New("id-3", "permission-3", false))

	assert.True(t, result.HasRoleID("id-1"))
	assert.True(t, result.HasRoleID("id-2"))
	assert.True(t, result.HasRoleID("id-3"))
	assert.False(t, result.HasRoleID("id-4"))
}

func assertPermissionsInPermissions(t assert.TestingT, actual permissions.Permissions, expected []naisiov1.AccessPolicyPermission) {
	for _, v := range expected {
		assertPermissionInPermissions(t, actual, string(v))
	}
}

func assertPermissionInPermissions(t assert.TestingT, permissions permissions.Permissions, element string) {
	assert.Contains(t, permissions, element)
	assert.Equal(t, element, permissions[element].Name)
	assert.True(t, permissions[element].Enabled)
	assert.NotEmpty(t, permissions[element].ID)
}

func assertPermissionIDsMatch(t assert.TestingT, app *msgraph.Application, extracted permissions.Permissions) {
	scopes := app.API.OAuth2PermissionScopes
	roles := app.AppRoles

	for key, value := range extracted {
		for _, scope := range scopes {
			if *scope.Value == key {
				assert.Equalf(t, *scope.ID, value.ID, fmt.Sprintf("expected equal UUID for scope %s", key))
			}
		}

		for _, role := range roles {
			if *role.Value == key {
				assert.Equalf(t, *role.ID, value.ID, fmt.Sprintf("expected equal UUID for role %s", key))
			}
		}
	}
}

func minimalApplication() *naisiov1.AzureAdApplication {
	return &naisiov1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: naisiov1.AzureAdApplicationSpec{
			SecretName: "test",
		},
	}
}

func minimalMsGraphApplication() *msgraph.Application {
	commonPermission := permissions.NewGenerateIdEnabled("common")
	commonPermission2 := permissions.NewGenerateIdEnabled("common-2")

	return &msgraph.Application{
		API: &msgraph.APIApplication{
			OAuth2PermissionScopes: []msgraph.PermissionScope{
				permissionscope.NewGenerateId("scope-1"),
				permissionscope.NewGenerateId("scope-2"),
				permissionscope.New(commonPermission.ID, commonPermission.Name),
				permissionscope.New(commonPermission2.ID, commonPermission2.Name),
			},
		},
		AppRoles: []msgraph.AppRole{
			approle.NewGenerateId("role-1"),
			approle.NewGenerateId("role-2"),
			approle.NewGenerateId("role-3"),
			approle.New(commonPermission.ID, commonPermission.Name),
			approle.New(commonPermission2.ID, commonPermission2.Name),
		},
	}
}

func legacyMsGraphApplication() *msgraph.Application {
	return &msgraph.Application{
		API: &msgraph.APIApplication{
			OAuth2PermissionScopes: []msgraph.PermissionScope{
				permissionscope.NewGenerateId(permissions.DefaultPermissionScopeValue),
			},
		},
		AppRoles: []msgraph.AppRole{
			approle.NewGenerateId(permissions.DefaultAppRoleValue),
		},
	}
}

func assertContainsDefaultPermissions(t assert.TestingT, desired permissions.Permissions) {
	all := permissions.PermissionList{
		permissions.FromAppRole(approle.DefaultRole()),
	}

	for _, permission := range all {
		assert.Contains(t, desired, permission.Name)
		assert.Equal(t, permission.Name, desired[permission.Name].Name)
		assert.Equal(t, permission.Enabled, desired[permission.Name].Enabled)
		assert.Equal(t, permission.ID, desired[permission.Name].ID)
	}
}
