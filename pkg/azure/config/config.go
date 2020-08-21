package config

import (
	"fmt"

	flag "github.com/spf13/pflag"
)

type Config struct {
	Auth                      Auth            `json:"auth"`
	Tenant                    string          `json:"tenant"`
	PermissionGrantResourceId string          `json:"permissionGrantResourceId"`
	TeamsManagement           TeamsManagement `json:"teamsManagement"`
}

type Auth struct {
	ClientId     string `json:"client-id"`
	ClientSecret string `json:"client-secret"`
}

type TeamsManagement struct {
	ServicePrincipalId string `json:"service-principal-id"`
}

const (
	ClientId                          = "azure.auth.client-id"
	ClientSecret                      = "azure.auth.client-secret"
	Tenant                            = "azure.tenant"
	PermissionGrantResourceId         = "azure.permissiongrantresourceid"
	TeamsManagementServicePrincipalId = "azure.teamsmanagement.service-principal-id"
	wellKnownUrlFormat                = "https://login.microsoftonline.com/%s/v2.0/.well-known/openid-configuration"
)

func SetupFlags() {
	flag.String(ClientId, "", "Client ID for Azure AD authentication")
	flag.String(ClientSecret, "", "Client secret for Azure AD authentication")
	flag.String(Tenant, "", "Tenant for Azure AD")
	flag.String(PermissionGrantResourceId, "", "Object ID for Graph API permissions grant ('GraphAggregatorService' in Enterprise Applications)")
	flag.String(TeamsManagementServicePrincipalId, "", "Service Principal ID for teams management application containing team groups")
}

func WellKnownUrl(tenant string) string {
	return fmt.Sprintf(wellKnownUrlFormat, tenant)
}
