package preauthorizedapp

import (
	"testing"

	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/permissions"
	"github.com/nais/azureator/pkg/azure/resource"
)

func TestList_HasResource(t *testing.T) {
	preAuthApps := List([]msgraph.PreAuthorizedApplication{
		{
			AppID: new("app-1"),
			DelegatedPermissionIDs: []string{
				permissions.DefaultPermissionScopeId,
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
