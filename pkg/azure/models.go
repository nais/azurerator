package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
)

type Client interface {
	RegisterOrUpdateApplication(credential v1alpha1.AzureAdCredential) (Credentials, error)
	DeleteApplication(credential v1alpha1.AzureAdCredential) error
}

type client struct {
	ctx                    context.Context
	config                 *Config
	servicePrincipalClient graphrbac.ServicePrincipalsClient
	applicationsClient     graphrbac.ApplicationsClient
}

type Credentials struct {
	Public  Public  `json:"public"`
	Private Private `json:"private"`
}

type Public struct {
	ClientId string `json:"clientId"`
	Key      Key    `json:"key"`
}

type Private struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Key          Key    `json:"key"`
}

type Key struct {
	KeyBase64 string `json:"keyBase64"`
	KeyId     string `json:"keyId"`
}
