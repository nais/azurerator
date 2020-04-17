package client

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/nais/azureator/pkg/azure"
	gocache "github.com/patrickmn/go-cache"
)

type client struct {
	ctx                    context.Context
	config                 *azure.Config
	servicePrincipalClient graphrbac.ServicePrincipalsClient
	applicationsClient     graphrbac.ApplicationsClient
	applicationsCache      gocache.Cache
}

const (
	IntegratedAppTag string = "WindowsAzureActiveDirectoryIntegratedApp"
	SignInAudience   string = "AzureADMyOrg"
)

func NewClient(ctx context.Context, cfg *azure.Config) (azure.Client, error) {
	spClient, err := getServicePrincipalsClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate service principal client: %w", err)
	}

	appClient, err := getApplicationsClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate applications client: %w", err)
	}
	cache := *gocache.New(gocache.NoExpiration, gocache.NoExpiration)
	return newClient(ctx, cfg, spClient, appClient, cache), nil
}

func getServicePrincipalsClient(cfg *azure.Config) (graphrbac.ServicePrincipalsClient, error) {
	spClient := graphrbac.NewServicePrincipalsClient(cfg.Tenant)
	a, err := azure.GetGraphAuthorizer(cfg)
	if err != nil {
		return spClient, fmt.Errorf("failed to get graph authorizer: %w", err)
	}
	spClient.Authorizer = a
	return spClient, nil
}

func getApplicationsClient(cfg *azure.Config) (graphrbac.ApplicationsClient, error) {
	appClient := graphrbac.NewApplicationsClient(cfg.Tenant)
	a, err := azure.GetGraphAuthorizer(cfg)
	if err != nil {
		return appClient, fmt.Errorf("failed to get graph authorizer: %w", err)
	}
	appClient.Authorizer = a
	return appClient, nil
}

func newClient(ctx context.Context, cfg *azure.Config, spClient graphrbac.ServicePrincipalsClient, appClient graphrbac.ApplicationsClient, cache gocache.Cache) client {
	return client{
		ctx:                    ctx,
		config:                 cfg,
		servicePrincipalClient: spClient,
		applicationsClient:     appClient,
		applicationsCache:      cache,
	}
}
