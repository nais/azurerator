package azure

import (
	"github.com/nais/azureator/pkg/azure/util/permissions"
)

type ApplicationResult struct {
	ClientId           string                  `json:"clientId"`
	ObjectId           string                  `json:"objectId"`
	ServicePrincipalId string                  `json:"servicePrincipalId"`
	Permissions        permissions.Permissions `json:"permissions"`
	PreAuthorizedApps  PreAuthorizedApps       `json:"preAuthorizedApps"`
	Tenant             string                  `json:"tenant"`
	Result             OperationResult         `json:"result"`
}

func (a ApplicationResult) IsNotModified() bool {
	return a.Result == OperationResultNotModified
}
