package azure

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
)

var graphAuthorizer autorest.Authorizer

// GetGraphAuthorizer gets an OAuthTokenAuthorizer for graphrbac API.
func GetGraphAuthorizer(cfg *Config) (autorest.Authorizer, error) {
	if graphAuthorizer != nil {
		return graphAuthorizer, nil
	}

	var a autorest.Authorizer
	var err error

	oauthConfig, err := adal.NewOAuthConfig(cfg.Endpoints.Directory, cfg.Tenant)
	if err != nil {
		return nil, err
	}

	token, err := adal.NewServicePrincipalToken(*oauthConfig, cfg.Auth.ClientId, cfg.Auth.ClientSecret, cfg.Endpoints.Graph)
	if err != nil {
		return nil, err
	}
	a = autorest.NewBearerAuthorizer(token)

	// cache
	graphAuthorizer = a

	return graphAuthorizer, err
}
