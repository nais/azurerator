package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
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

func (c client) servicePrincipalExists(tx azure.Transaction) (bool, msgraphbeta.ServicePrincipal, error) {
	clientId := tx.Resource.Status.ClientId
	r := c.graphBetaClient.ServicePrincipals().Request()
	r.Filter(util.FilterByAppId(clientId))
	sps, err := r.GetN(tx.Ctx, 1000)
	if err != nil {
		return false, msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to lookup service principal: %w", err)
	}
	if len(sps) == 0 {
		return false, msgraphbeta.ServicePrincipal{}, nil
	}
	return true, sps[0], nil
}

func (c client) upsertServicePrincipal(tx azure.Transaction) (msgraphbeta.ServicePrincipal, error) {
	exists, sp, err := c.servicePrincipalExists(tx)
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, err
	}
	if exists {
		return sp, nil
	}
	application := msgraph.Application{AppID: &tx.Resource.Status.ClientId}
	sp, err = c.registerServicePrincipal(tx.Ctx, application)
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
