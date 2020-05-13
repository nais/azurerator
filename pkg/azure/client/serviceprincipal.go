package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
)

func (c client) registerServicePrincipal(ctx context.Context, id azure.ClientId) (msgraphbeta.ServicePrincipal, error) {
	servicePrincipal, err := c.graphBetaClient.ServicePrincipals().Request().Add(ctx, toServicePrincipal(id))
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to register service principal: %w", err)
	}
	return *servicePrincipal, nil
}

func (c client) servicePrincipalExists(ctx context.Context, id azure.ClientId) (bool, msgraphbeta.ServicePrincipal, error) {
	r := c.graphBetaClient.ServicePrincipals().Request()
	r.Filter(util.FilterByAppId(id))
	sps, err := r.GetN(ctx, 1000)
	if err != nil {
		return false, msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to lookup service principal: %w", err)
	}
	if len(sps) == 0 {
		return false, msgraphbeta.ServicePrincipal{}, nil
	}
	return true, sps[0], nil
}

func (c client) getServicePrincipalsWithFilter(ctx context.Context, filter azure.Filter) ([]msgraphbeta.ServicePrincipal, error) {
	r := c.graphBetaClient.ServicePrincipals().Request()
	r.Filter(filter)
	sps, err := r.GetN(ctx, 1000)
	if err != nil {
		return []msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to lookup service principals with filter '%s': %w", filter, err)
	}
	return sps, nil
}

func toServicePrincipal(clientId azure.ClientId) *msgraphbeta.ServicePrincipal {
	return &msgraphbeta.ServicePrincipal{
		AppID:                     &clientId,
		AppRoleAssignmentRequired: ptr.Bool(false),
	}
}
