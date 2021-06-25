package preauthorizedapp

import (
	"testing"

	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/client/application/permissionscope"
	"github.com/nais/azureator/pkg/azure/resource"
)

func TestList_HasResource(t *testing.T) {
	preAuthApps := List([]msgraph.PreAuthorizedApplication{
		{
			AppID: ptr.String("app-1"),
			DelegatedPermissionIDs: []string{
				permissionscope.DefaultAccessScopeId,
			},
		},
	})

	expected := resource.Resource{
		ClientId: "app-1",
	}
	assert.True(t, preAuthApps.HasResource(expected))

	expected = resource.Resource{
		ClientId: "app-2",
	}
	assert.False(t, preAuthApps.HasResource(expected))
}
