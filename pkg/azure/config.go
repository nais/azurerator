package azure

import (
	flag "github.com/spf13/pflag"
)

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
	ClientId                  = "azure.auth.clientId"
	ClientSecret              = "azure.auth.clientSecret"
	Tenant                    = "azure.tenant"
	EndpointsGraph            = "azure.endpoints.graph"
	EndpointsDirectory        = "azure.endpoints.directory"
	PermissionGrantResourceId = "azure.permissiongrantresourceid"
)

func SetupFlags() {
	flag.String(ClientId, "", "Client ID for Azure AD authentication")
	flag.String(ClientSecret, "", "Client secret for Azure AD authentication")
	flag.String(Tenant, "", "Tenant for Azure AD")
	flag.String(EndpointsGraph, "", "Endpoint to Graph API")
	flag.String(EndpointsDirectory, "", "Endpoint to Azure AD")
	flag.String(PermissionGrantResourceId, "", "Resource ID for permissions grant")
}
