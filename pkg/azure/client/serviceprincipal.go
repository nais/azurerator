package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

func (c client) registerServicePrincipal(ctx context.Context, application msgraph.Application) (msgraphbeta.ServicePrincipal, error) {
	servicePrincipal, err := c.graphBetaClient.ServicePrincipals().Request().Add(ctx, toServicePrincipal(application))
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to register service principal: %w", err)
	}
	return *servicePrincipal, nil
}

func (c client) servicePrincipalExists(ctx context.Context, credential v1alpha1.AzureAdCredential) (bool, msgraphbeta.ServicePrincipal, error) {
	clientId := credential.Status.ClientId
	r := c.graphBetaClient.ServicePrincipals().Request()
	r.Filter(util.FilterByAppId(clientId))
	sps, err := r.GetN(ctx, 1000)
	if err != nil {
		return false, msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to lookup service principal: %w", err)
	}
	if len(sps) == 0 {
		return false, msgraphbeta.ServicePrincipal{}, nil
	}
	return true, sps[0], nil
}

func (c client) upsertServicePrincipal(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraphbeta.ServicePrincipal, error) {
	exists, sp, err := c.servicePrincipalExists(ctx, credential)
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, err
	}
	if exists {
		return sp, nil
	}
	application := msgraph.Application{AppID: &credential.Status.ClientId}
	sp, err = c.registerServicePrincipal(ctx, application)
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, err
	}
	return sp, err
}

func toServicePrincipal(application msgraph.Application) *msgraphbeta.ServicePrincipal {
	return &msgraphbeta.ServicePrincipal{
		AppID: application.AppID,
	}
}
