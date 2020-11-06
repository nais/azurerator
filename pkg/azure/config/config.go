package config

import (
	"fmt"

	flag "github.com/spf13/pflag"
)

type Config struct {
	Auth                      Auth                `json:"auth"`
	Tenant                    string              `json:"tenant"`
	TenantName                string              `json:"tenant-name"`
	PermissionGrantResourceId string              `json:"permissionGrantResourceId"`
	TeamsManagement           TeamsManagement     `json:"teamsManagement"`
	ClaimsMappingPolicy       ClaimsMappingPolicy `json:"claims-mapping-policy"`
}

type Auth struct {
	ClientId     string `json:"client-id"`
	ClientSecret string `json:"client-secret"`
}

type TeamsManagement struct {
	ServicePrincipalId string `json:"service-principal-id"`
}

type ClaimsMappingPolicy struct {
	NavIdent string `json:"navident"`
}

const (
	ClientId                          = "azure.auth.client-id"
	ClientSecret                      = "azure.auth.client-secret"
	Tenant                            = "azure.tenant"
	TenantName                        = "azure.tenant-name"
	PermissionGrantResourceId         = "azure.permissiongrantresourceid"
	ClaimsMappingPoliciesNavIdent     = "azure.claims-mapping-policy.navident"
	TeamsManagementServicePrincipalId = "azure.teamsmanagement.service-principal-id"
	wellKnownUrlFormat                = "https://login.microsoftonline.com/%s/v2.0/.well-known/openid-configuration"
)

func SetupFlags() {
	flag.String(ClientId, "", "Client ID for Azure AD authentication")
	flag.String(ClientSecret, "", "Client secret for Azure AD authentication")
	flag.String(Tenant, "", "Tenant for Azure AD")
	flag.String(TenantName, "", "Alias/name of tenant for Azure AD")
	flag.String(PermissionGrantResourceId, "", "Object ID for Graph API permissions grant ('GraphAggregatorService' in Enterprise Applications)")
	flag.String(ClaimsMappingPoliciesNavIdent, "", "Claims-mapping policy ID for NavIdent")
	flag.String(TeamsManagementServicePrincipalId, "", "Service Principal ID for teams management application containing team groups")
}

func WellKnownUrl(tenant string) string {
	return fmt.Sprintf(wellKnownUrlFormat, tenant)
}
