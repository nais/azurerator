package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/msauth"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"golang.org/x/oauth2"
)

type client struct {
	config          *azure.Config
	graphClient     *msgraph.GraphServiceRequestBuilder
	graphBetaClient *msgraphbeta.GraphServiceRequestBuilder
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

	return client{
		config:          cfg,
		graphClient:     graphClient,
		graphBetaClient: graphBetaClient,
	}, nil
}
