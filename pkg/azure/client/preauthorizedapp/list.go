package preauthorizedapp

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/resource"
)

type List []msgraph.PreAuthorizedApplication

func (l List) HasResource(resource resource.Resource) bool {
	for _, app := range l {
		if *app.AppID == resource.ClientId {
			return true
		}
	}
	return false
}
