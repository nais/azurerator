package config

import (
	flag "github.com/spf13/pflag"
)

type Config struct {
	Auth                      Auth   `json:"auth"`
	Tenant                    string `json:"tenant"`
	PermissionGrantResourceId string `json:"permissionGrantResourceId"`
}

type Auth struct {
	ClientId     string `json:"client-id"`
	ClientSecret string `json:"client-secret"`
}

const (
	ClientId                  = "azure.auth.client-id"
	ClientSecret              = "azure.auth.client-secret"
	Tenant                    = "azure.tenant"
	PermissionGrantResourceId = "azure.permissiongrantresourceid"
)

func SetupFlags() {
	flag.String(ClientId, "", "Client ID for Azure AD authentication")
	flag.String(ClientSecret, "", "Client secret for Azure AD authentication")
	flag.String(Tenant, "", "Tenant for Azure AD")
	flag.String(PermissionGrantResourceId, "", "Resource ID for permissions grant")
}
