package azure

type Config struct {
	Auth                      Auth      `json:"auth"`
	Endpoints                 Endpoints `json:"endpoints"`
	Tenant                    string    `json:"tenant"`
	PermissionGrantResourceId string    `json:"permissionGrantResourceId"`
}

type Auth struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type Endpoints struct {
	Directory string `json:"directory"`
	Graph     string `json:"graph"`
}

const (
	ClientId                  = "azuread.auth.client.id"
	ClientSecret              = "azuread.auth.client.secret"
	Tenant                    = "azuread.tenant"
	EndpointsGraph            = "azuread.endpoints.graph"
	EndpointsDirectory        = "azuread.endpoints.directory"
	PermissionGrantResourceId = "azuread.permissiongrantresourceid"
)
