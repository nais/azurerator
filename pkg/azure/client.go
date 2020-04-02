package azure

import (
	"github.com/Azure/azure-sdk-for-go/profiles/latest/graphrbac/graphrbac"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
)

type Client interface {
	CreateOrUpdateApplication(azureAdCredential v1alpha1.AzureAdCredential) Credentials
	DeleteApplication(azureAdCredential v1alpha1.AzureAdCredential)
}

type client struct {
	config                 *Config
	servicePrincipalClient graphrbac.ServicePrincipalsClient
	applicationsClient     graphrbac.ApplicationsClient
}

type Credentials struct {
	Public  Public  `json:"public"`
	Private Private `json:"private"`
}

type Public struct {
	ClientId     string `json:"clientId"`
	Key          Key    `json:"key"`
}

type Private struct {
	ClientId string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Key      Key    `json:"key"`
}

type Key struct {
	KeyBase64 string `json:"keyBase64"`
	KeyId     string `json:"keyId"`
}

const (
	IntegratedAppTag string = "WindowsAzureActiveDirectoryIntegratedApp"
)

func NewClient(cfg *Config) (Client, error) {
	spClient, err := getServicePrincipalsClient(cfg)
	if err != nil {
		return nil, err
	}

	appClient, err := getApplicationsClient(cfg)
	if err != nil {
		return nil, err
	}

	return newClient(cfg, spClient, appClient), nil
}

func getServicePrincipalsClient(cfg *Config) (graphrbac.ServicePrincipalsClient, error) {
	spClient := graphrbac.NewServicePrincipalsClient(cfg.Tenant)
	a, err := GetGraphAuthorizer(cfg)
	if err != nil {
		return spClient, err
	}
	spClient.Authorizer = a
	return spClient, nil
}

func getApplicationsClient(cfg *Config) (graphrbac.ApplicationsClient, error) {
	appClient := graphrbac.NewApplicationsClient(cfg.Tenant)
	a, err := GetGraphAuthorizer(cfg)
	if err != nil {
		return appClient, err
	}
	appClient.Authorizer = a
	return appClient, nil
}

func newClient(cfg *Config, spClient graphrbac.ServicePrincipalsClient, appClient graphrbac.ApplicationsClient) Client {
	return client{
		config:                 cfg,
		servicePrincipalClient: spClient,
		applicationsClient:     appClient,
	}
}

func (c client) CreateOrUpdateApplication(azureAdCredential v1alpha1.AzureAdCredential) Credentials {
	// TODO
	return Credentials{}
}

func (c client) DeleteApplication(azureAdCredential v1alpha1.AzureAdCredential) {
	// TODO
}
