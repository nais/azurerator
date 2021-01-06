package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
)

type servicePrincipal struct {
	client
}

func (c client) servicePrincipal() servicePrincipal {
	return servicePrincipal{c}
}

func (s servicePrincipal) register(ctx context.Context, id azure.ClientId) (msgraphbeta.ServicePrincipal, error) {
	request := &msgraphbeta.ServicePrincipal{
		AppID:                     &id,
		AppRoleAssignmentRequired: ptr.Bool(false),
	}
	servicePrincipal, err := s.graphBetaClient.ServicePrincipals().Request().Add(ctx, request)
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to register service principal: %w", err)
	}
	return *servicePrincipal, nil
}

func (s servicePrincipal) exists(ctx context.Context, id azure.ClientId) (bool, msgraphbeta.ServicePrincipal, error) {
	r := s.graphBetaClient.ServicePrincipals().Request()
	r.Filter(util.FilterByAppId(id))
	sps, err := r.GetN(ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return false, msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to lookup service principal: %w", err)
	}
	if len(sps) == 0 {
		return false, msgraphbeta.ServicePrincipal{}, nil
	}
	return true, sps[0], nil
}

func (s servicePrincipal) getWithFilter(ctx context.Context, filter azure.Filter) ([]msgraphbeta.ServicePrincipal, error) {
	r := s.graphBetaClient.ServicePrincipals().Request()
	r.Filter(filter)
	sps, err := r.GetN(ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return []msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to lookup service principals with filter '%s': %w", filter, err)
	}
	return sps, nil
}
