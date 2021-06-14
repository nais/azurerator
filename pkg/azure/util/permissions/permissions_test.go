package permissions_test

import (
	"fmt"
	"testing"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/azureator/pkg/azure/util/approle"
	"github.com/nais/azureator/pkg/azure/util/permissions"
	"github.com/nais/azureator/pkg/azure/util/permissionscope"
)

func TestGenerateDesiredPermissionSet(t *testing.T) {
	app := minimalApplication()
	desired := permissions.GenerateDesiredPermissionSet(*app)
	assert.Empty(t, desired)

	app.Spec.PreAuthorizedApplications = []naisiov1.AccessPolicyRule{
		{
			Application: "a",
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles:  []string{"read", "write"},
				Scopes: []string{"admin"},
			},
		}, {
			Application: "b",
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles:  []string{"read"},
				Scopes: []string{"write"},
			},
		}, {
			Application: "c",
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles:  []string{"write", "admin"},
				Scopes: []string{"read"},
			},
		},
	}

	desired = permissions.GenerateDesiredPermissionSet(*app)
	assert.Len(t, desired, 3)
	assertPermissionsInPermissions(t, desired, []string{"read", "write", "admin"})
}

func TestExtractPermissions(t *testing.T) {
	msgraphApp := minimalMsGraphApplication()
	actual := permissions.ExtractPermissions(msgraphApp)
	expected := []string{
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
	app.Spec.PreAuthorizedApplications = []naisiov1.AccessPolicyRule{
		{
			Application: "a",
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles:  []string{"role-1"},
				Scopes: []string{"scope-1", "common"},
			},
		}, {
			Application: "b",
			Permissions: &naisiov1.AccessPolicyPermissions{
				Roles:  []string{"role-3", "common"},
				Scopes: []string{"scope-2", "scope-3"},
			},
		},
	}

	desired := permissions.GenerateDesiredPermissionSetPreserveExisting(*app, *existing)
	expected := []string{"role-1", "role-3", "scope-1", "scope-2", "scope-3", "common"}

	// length of desired set is set of roles + set of scopes in .Spec.PreAuthorizedApplications
	assert.Len(t, desired, len(expected))

	assertPermissionsInPermissions(t, desired, expected)
	assertPermissionIDsMatch(t, existing, desired)

	// assert that non-desired role "role-2" is removed
	assert.NotContains(t, desired, "role-2")
	// assert that non-desired scope and role "common-2" is removed
	assert.NotContains(t, desired, "common-2")
}

func assertPermissionsInPermissions(t assert.TestingT, actual permissions.Permissions, expected []string) {
	for _, v := range expected {
		assertPermissionInPermissions(t, actual, v)
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
