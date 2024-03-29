package result

import (
	"github.com/nais/azureator/pkg/azure/permissions"
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

func (a Application) IsCreated() bool {
	return a.Result == OperationCreated
}

func (a Application) IsUpdated() bool {
	return a.Result == OperationUpdated
}

func (a Application) IsModified() bool {
	return a.IsCreated() || a.IsUpdated()
}
