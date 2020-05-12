package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

func (c client) registerServicePrincipal(ctx context.Context, application msgraph.Application) (msgraphbeta.ServicePrincipal, error) {
	servicePrincipal, err := c.graphBetaClient.ServicePrincipals().Request().Add(ctx, toServicePrincipal(application))
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to register service principal: %w", err)
	}
	return *servicePrincipal, nil
}

func (c client) servicePrincipalExists(ctx context.Context, clientId string) (bool, msgraphbeta.ServicePrincipal, error) {
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

func (c client) upsertServicePrincipal(tx azure.Transaction) (msgraphbeta.ServicePrincipal, error) {
	clientId := tx.Resource.Status.ClientId
	exists, sp, err := c.servicePrincipalExists(tx.Ctx, clientId)
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

func (c client) getServicePrincipalId(ctx context.Context, clientId string) (string, error) {
	exists, sp, err := c.servicePrincipalExists(ctx, clientId)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("service principal does not exist for app with id '%s': %w", clientId, err)
	}
	return *sp.ID, nil
}

func toServicePrincipal(application msgraph.Application) *msgraphbeta.ServicePrincipal {
	return &msgraphbeta.ServicePrincipal{
		AppID:                     application.AppID,
		AppRoleAssignmentRequired: ptr.Bool(true),
	}
}
