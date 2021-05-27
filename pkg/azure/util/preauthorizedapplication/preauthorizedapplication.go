package preauthorizedapplication

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
)

func ResourceInPreAuthorizedApps(resource azure.Resource, apps []msgraph.PreAuthorizedApplication) bool {
	for _, app := range apps {
		if *app.AppID == resource.ClientId {
			return true
		}
	}
	return false
}
