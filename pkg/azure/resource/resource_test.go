package resource_test

import (
	"testing"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/client/application/approle"
	"github.com/nais/azureator/pkg/azure/client/application/permissionscope"
	"github.com/nais/azureator/pkg/azure/permissions"
	"github.com/nais/azureator/pkg/azure/resource"
)

func TestResource_ToPreAuthorizedApp(t *testing.T) {
	app := resource.Resource{
		ClientId: "app-1",
		AccessPolicyInboundRule: naisiov1.AccessPolicyInboundRule{
			Permissions: &naisiov1.AccessPolicyPermissions{
				Scopes: []naisiov1.AccessPolicyPermission{
					"scope-1",
					"scope-2",
				},
			},
		},
	}

	existingPermissions := make(permissions.Permissions)
	existingPermissions.Add(permissions.FromPermissionScope(permissionscope.DefaultScope()))
	existingPermissions.Add(permissions.NewGenerateIdEnabled("scope-1"))

	preAuthorizedApp := app.ToPreAuthorizedApp(existingPermissions)
	expected := msgraph.PreAuthorizedApplication{
		AppID: ptr.String("app-1"),
		DelegatedPermissionIDs: []string{
			permissionscope.DefaultAccessScopeId,
			string(existingPermissions["scope-1"].ID),
		},
	}

	assert.Equal(t, expected.AppID, preAuthorizedApp.AppID)
	assert.ElementsMatch(t, expected.DelegatedPermissionIDs, preAuthorizedApp.DelegatedPermissionIDs)
}

func TestResource_ToAppRoleAssignment(t *testing.T) {
	app := resource.Resource{
		Name:          "app-1",
		ObjectId:      "app-1-object-id",
		PrincipalType: resource.PrincipalTypeServicePrincipal,
	}
	target := "target-object-id"
	permission := permissions.NewGenerateIdEnabled("scope-1")

	appRoleAssignment := app.ToAppRoleAssignment(target, permission)
	expected := msgraph.AppRoleAssignment{
		AppRoleID:            &permission.ID,
		PrincipalID:          (*msgraph.UUID)(ptr.String(app.ObjectId)),
		PrincipalDisplayName: ptr.String("app-1"),
		PrincipalType:        ptr.String(string(resource.PrincipalTypeServicePrincipal)),
		ResourceID:           (*msgraph.UUID)(ptr.String(target)),
	}

	assert.Equal(t, expected, appRoleAssignment)
}

func TestResources_FilterByRole(t *testing.T) {
	role := permissions.NewGenerateIdEnabled("some-role")

	resourceWithNoPermissions := resource.Resource{
		Name: "app-1",
	}
	resourceWithScopes := resource.Resource{
		Name: "app-2",
		AccessPolicyInboundRule: naisiov1.AccessPolicyInboundRule{
			Permissions: &naisiov1.AccessPolicyPermissions{
				Scopes: []naisiov1.AccessPolicyPermission{"some-scope", "common"},
			},
		},
	}
	resourceWithRoles := resource.Resource{
		Name: "app-3",
		AccessPolicyInboundRule: naisiov1.AccessPolicyInboundRule{
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles: []naisiov1.AccessPolicyPermission{"some-role", "common"},
			},
		},
	}
	resourceWithScopesAndRoles := resource.Resource{
		Name: "app-4",
		AccessPolicyInboundRule: naisiov1.AccessPolicyInboundRule{
			Permissions: &naisiov1.AccessPolicyPermissions{
				Scopes: []naisiov1.AccessPolicyPermission{"some-scope", "common"},
				Roles:  []naisiov1.AccessPolicyPermission{"some-role", "common"},
			},
		},
	}
	resourceWithDuplicatePermissions := resource.Resource{
		Name: "app-5",
		AccessPolicyInboundRule: naisiov1.AccessPolicyInboundRule{
			Permissions: &naisiov1.AccessPolicyPermissions{
				Scopes: []naisiov1.AccessPolicyPermission{"some-scope", "some-scope"},
				Roles:  []naisiov1.AccessPolicyPermission{"some-role", "some-role"},
			},
		},
	}

	t.Run("no permissions should return empty", func(t *testing.T) {
		resources := resource.Resources{
			resourceWithNoPermissions,
		}
		filtered := resources.FilterByRole(role)

		assert.Empty(t, filtered)
		assert.NotContains(t, filtered, resourceWithNoPermissions)
	})

	t.Run("only desires scopes should return empty", func(t *testing.T) {
		resources := resource.Resources{
			resourceWithScopes,
		}
		filtered := resources.FilterByRole(role)

		assert.Empty(t, filtered)
		assert.NotContains(t, filtered, resourceWithNoPermissions)
	})

	t.Run("only desires matching role should return match", func(t *testing.T) {
		resources := resource.Resources{
			resourceWithRoles,
		}
		filtered := resources.FilterByRole(role)

		assert.Len(t, filtered, 1)
		assert.Contains(t, filtered, resourceWithRoles)
	})

	t.Run("matching role and scope should return match", func(t *testing.T) {
		resources := resource.Resources{
			resourceWithScopesAndRoles,
		}
		filtered := resources.FilterByRole(role)

		assert.Len(t, filtered, 1)
		assert.Contains(t, filtered, resourceWithScopesAndRoles)
	})

	t.Run("duplicate desired roles should only return single match", func(t *testing.T) {
		resources := resource.Resources{
			resourceWithDuplicatePermissions,
		}
		filtered := resources.FilterByRole(role)

		assert.Len(t, filtered, 1)
		assert.Contains(t, filtered, resourceWithDuplicatePermissions)
	})

	t.Run("combination should only return assignees with matching roles", func(t *testing.T) {
		resources := resource.Resources{
			resourceWithNoPermissions,
			resourceWithScopes,
			resourceWithRoles,
			resourceWithScopesAndRoles,
			resourceWithDuplicatePermissions,
		}
		filtered := resources.FilterByRole(role)

		assert.Len(t, filtered, 3)
		assert.Contains(t, filtered, resourceWithRoles)
		assert.Contains(t, filtered, resourceWithScopesAndRoles)
		assert.Contains(t, filtered, resourceWithDuplicatePermissions)
		assert.NotContains(t, filtered, resourceWithNoPermissions)
		assert.NotContains(t, filtered, resourceWithScopes)
	})
}

func TestResources_FilterByPrincipalType(t *testing.T) {
	principalType := resource.PrincipalTypeServicePrincipal

	resources := resource.Resources{
		resource.Resource{
			Name:          "app-1",
			PrincipalType: resource.PrincipalTypeServicePrincipal,
		},
		resource.Resource{
			Name:          "group-1",
			PrincipalType: resource.PrincipalTypeGroup,
		},
		resource.Resource{
			Name:          "user-1",
			PrincipalType: resource.PrincipalTypeUser,
		},
	}

	filtered := resources.FilterByPrincipalType(principalType)

	assert.Len(t, filtered, 1)

	for _, res := range filtered {
		assert.Equal(t, principalType, res.PrincipalType)
	}
}

func TestResources_ExtractDesiredAssignees(t *testing.T) {
	app1 := resource.Resource{
		Name:          "app-1",
		PrincipalType: resource.PrincipalTypeServicePrincipal,
		AccessPolicyInboundRule: naisiov1.AccessPolicyInboundRule{
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles: []naisiov1.AccessPolicyPermission{"some-permission"},
			},
		},
	}
	app2 := resource.Resource{
		Name:          "app-2",
		PrincipalType: resource.PrincipalTypeServicePrincipal,
		AccessPolicyInboundRule: naisiov1.AccessPolicyInboundRule{
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles: []naisiov1.AccessPolicyPermission{"some-other-permission"},
			},
		},
	}
	app3 := resource.Resource{
		Name:          "app-2",
		PrincipalType: resource.PrincipalTypeServicePrincipal,
	}
	app4 := resource.Resource{
		Name:          "app-4",
		PrincipalType: resource.PrincipalTypeServicePrincipal,
		AccessPolicyInboundRule: naisiov1.AccessPolicyInboundRule{
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles: []naisiov1.AccessPolicyPermission{"some-permission", "some-permission"},
			},
		},
	}
	group1 := resource.Resource{
		Name:          "group-1",
		PrincipalType: resource.PrincipalTypeGroup,
	}
	group2 := resource.Resource{
		Name:          "group-2",
		PrincipalType: resource.PrincipalTypeGroup,
	}

	assignees := resource.Resources{app1, app2, app3, group1, group2}

	t.Run("custom role for service principal should return all matching assignees", func(t *testing.T) {
		role := permissions.NewGenerateIdEnabled("some-permission")
		principalType := resource.PrincipalTypeServicePrincipal

		desired := assignees.FilterByPrincipalType(principalType).ExtractDesiredAssignees(principalType, role)
		assert.Len(t, desired, 1)
		assert.Contains(t, desired, app1)
		assert.NotContains(t, desired, app2)
		assert.NotContains(t, desired, app3)
	})

	t.Run("duplicate desired custom role for service principal should return all matching assignees without duplicates", func(t *testing.T) {
		role := permissions.NewGenerateIdEnabled("some-permission")
		principalType := resource.PrincipalTypeServicePrincipal
		assignees := resource.Resources{app4}

		desired := assignees.FilterByPrincipalType(principalType).ExtractDesiredAssignees(principalType, role)
		assert.Len(t, desired, 1)
		assert.Contains(t, desired, app4)
	})

	t.Run("default role for service principals should return all assignees", func(t *testing.T) {
		role := permissions.FromAppRole(approle.DefaultRole())
		principalType := resource.PrincipalTypeServicePrincipal

		desired := assignees.FilterByPrincipalType(principalType).ExtractDesiredAssignees(principalType, role)
		assert.Len(t, desired, 3)
		assert.Contains(t, desired, app1)
		assert.Contains(t, desired, app2)
		assert.Contains(t, desired, app3)
	})

	t.Run("custom role for groups should return empty", func(t *testing.T) {
		role := permissions.NewGenerateIdEnabled("some-permission")
		principalType := resource.PrincipalTypeGroup

		desired := assignees.FilterByPrincipalType(principalType).ExtractDesiredAssignees(principalType, role)
		assert.Empty(t, desired)
	})

	t.Run("default for groups should return all assignees", func(t *testing.T) {
		role := permissions.FromAppRole(approle.DefaultGroupRole())
		principalType := resource.PrincipalTypeGroup

		desired := assignees.FilterByPrincipalType(principalType).ExtractDesiredAssignees(principalType, role)
		assert.Len(t, desired, 2)
		assert.Contains(t, desired, group1)
		assert.Contains(t, desired, group2)
	})
}
