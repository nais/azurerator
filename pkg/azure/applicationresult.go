package azure

type ApplicationResult struct {
	ClientId           string            `json:"clientId"`
	ObjectId           string            `json:"objectId"`
	ServicePrincipalId string            `json:"servicePrincipalId"`
	PreAuthorizedApps  PreAuthorizedApps `json:"preAuthorizedApps"`
	Tenant             string            `json:"tenant"`
	Result             OperationResult   `json:"result"`
}

func (a ApplicationResult) IsNotModified() bool {
	return a.Result == OperationResultNotModified
}
