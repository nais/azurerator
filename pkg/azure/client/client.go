package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	gocache "github.com/patrickmn/go-cache"
	"github.com/yaegashi/msgraph.go/msauth"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"golang.org/x/oauth2"
)

type client struct {
	config            *azure.Config
	graphClient       *msgraph.GraphServiceRequestBuilder
	applicationsCache gocache.Cache
}

const (
	IntegratedAppTag string = "WindowsAzureActiveDirectoryIntegratedApp"
	SignInAudience   string = "AzureADMyOrg"
	IaCAppTag        string = "azurerator_appreg"
)

func New(ctx context.Context, cfg *azure.Config) (azure.Client, error) {
	m := msauth.NewManager()
	scopes := []string{msauth.DefaultMSGraphScope}
	ts, err := m.ClientCredentialsGrant(ctx, cfg.Tenant, cfg.Auth.ClientId, cfg.Auth.ClientSecret, scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate graph client: %w", err)
	}

	httpClient := oauth2.NewClient(ctx, ts)
	graphClient := msgraph.NewClient(httpClient)

	cache := *gocache.New(gocache.NoExpiration, gocache.NoExpiration)
	return client{
		config:            cfg,
		graphClient:       graphClient,
		applicationsCache: cache,
	}, nil
}