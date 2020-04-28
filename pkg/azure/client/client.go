package client

import (
	"context"
	"fmt"
	"time"

	"github.com/nais/azureator/pkg/azure"
	gocache "github.com/patrickmn/go-cache"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/msauth"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"golang.org/x/oauth2"
)

type client struct {
	config            *azure.Config
	graphClient       *msgraph.GraphServiceRequestBuilder
	graphBetaClient   *msgraphbeta.GraphServiceRequestBuilder
	applicationsCache gocache.Cache
}

func New(ctx context.Context, cfg *azure.Config) (azure.Client, error) {
	m := msauth.NewManager()
	scopes := []string{msauth.DefaultMSGraphScope}
	ts, err := m.ClientCredentialsGrant(ctx, cfg.Tenant, cfg.Auth.ClientId, cfg.Auth.ClientSecret, scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate graph client: %w", err)
	}

	httpClient := oauth2.NewClient(ctx, ts)
	graphClient := msgraph.NewClient(httpClient)
	graphBetaClient := msgraphbeta.NewClient(httpClient)

	defaultExpiration := 1 * time.Hour
	cleanupInterval := 30 * time.Minute

	cache := *gocache.New(defaultExpiration, cleanupInterval)
	return client{
		config:            cfg,
		graphClient:       graphClient,
		graphBetaClient:   graphBetaClient,
		applicationsCache: cache,
	}, nil
}
