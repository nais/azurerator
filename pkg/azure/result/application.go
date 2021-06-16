package result

import (
	"github.com/nais/azureator/pkg/azure/util/permissions"
)

type Application struct {
	ClientId           string                  `json:"clientId"`
	ObjectId           string                  `json:"objectId"`
	ServicePrincipalId string                  `json:"servicePrincipalId"`
	Permissions        permissions.Permissions `json:"permissions"`
	PreAuthorizedApps  PreAuthorizedApps       `json:"preAuthorizedApps"`
	Tenant             string                  `json:"tenant"`
	Result             Operation               `json:"result"`
}

func (a Application) IsNotModified() bool {
	return a.Result == OperationNotModified
}
