package client

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/nais/msgraph.go/msauth"
	"golang.org/x/oauth2"
	"google.golang.org/api/impersonate"

	"github.com/nais/azureator/pkg/config"
)

var (
	scopes = []string{msauth.DefaultMSGraphScope}
)

func NewClientCredentialsTokenSource(ctx context.Context, cfg *config.AzureConfig) (oauth2.TokenSource, error) {
	m := msauth.NewManager()
	ts, err := m.ClientCredentialsGrant(ctx, cfg.Tenant.Id, cfg.Auth.ClientId, cfg.Auth.ClientSecret, scopes)
	if err != nil {
		return nil, fmt.Errorf("performing client credentials grant: %w", err)
	}

	return ts, nil
}

type GoogleFederatedCredentialTokenSource struct {
	cred *azidentity.ClientAssertionCredential
	ctx  context.Context
	opts policy.TokenRequestOptions
}

func (in *GoogleFederatedCredentialTokenSource) Token() (*oauth2.Token, error) {
	tok, err := in.cred.GetToken(in.ctx, in.opts)
	if err != nil {
		return nil, fmt.Errorf("fetching azure token: %w", err)
	}

	return &oauth2.Token{
		AccessToken: tok.Token,
		TokenType:   "bearer",
		Expiry:      tok.ExpiresOn,
	}, nil
}

func NewGoogleFederatedCredentialsTokenSource(ctx context.Context, cfg *config.AzureConfig) (oauth2.TokenSource, error) {
	googleTokenSource, err := impersonate.IDTokenSource(ctx, impersonate.IDTokenConfig{
		Audience:        "api://AzureADTokenExchange",
		TargetPrincipal: fmt.Sprintf("azurerator@%s.iam.gserviceaccount.com", cfg.Auth.Google.ProjectID),
		IncludeEmail:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("creating google token source: %w", err)
	}

	googleAssertion := func(ctx context.Context) (string, error) {
		t, err := googleTokenSource.Token()
		if err != nil {
			return "", fmt.Errorf("fetching google token: %w", err)
		}

		return t.AccessToken, nil
	}

	cred, err := azidentity.NewClientAssertionCredential(cfg.Tenant.Id, cfg.Auth.ClientId, googleAssertion, nil)
	if err != nil {
		return nil, fmt.Errorf("creating azure assertion credential: %w", err)
	}

	ts := &GoogleFederatedCredentialTokenSource{
		cred: cred,
		ctx:  ctx,
		opts: policy.TokenRequestOptions{
			Scopes: scopes,
		},
	}

	return oauth2.ReuseTokenSource(nil, ts), nil
}
