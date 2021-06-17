package resource_test

import (
	"testing"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/client/application/permissionscope"
	"github.com/nais/azureator/pkg/azure/permissions"
	"github.com/nais/azureator/pkg/azure/resource"
)

func TestResource_ToPreAuthorizedApp(t *testing.T) {
	app := resource.Resource{
		ClientId: "app-1",
		AccessPolicyRule: naisiov1.AccessPolicyRule{
			Permissions: &naisiov1.AccessPolicyPermissions{
				Scopes: []string{
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

	assert.Equal(t, expected, preAuthorizedApp)
}
