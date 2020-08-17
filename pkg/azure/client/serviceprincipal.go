package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/azure/util/directoryobject"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type servicePrincipal struct {
	client
}

func (c client) servicePrincipal() servicePrincipal {
	return servicePrincipal{c}
}

func (s servicePrincipal) register(ctx context.Context, id azure.ClientId) (msgraphbeta.ServicePrincipal, error) {
	servicePrincipal, err := s.graphBetaClient.ServicePrincipals().Request().Add(ctx, s.toMsGraph(id))
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

func (s servicePrincipal) toMsGraph(clientId azure.ClientId) *msgraphbeta.ServicePrincipal {
	return &msgraphbeta.ServicePrincipal{
		AppID:                     &clientId,
		AppRoleAssignmentRequired: ptr.Bool(false),
	}
}

func (s servicePrincipal) getOwners(ctx context.Context, id azure.ServicePrincipalId) ([]msgraphbeta.DirectoryObject, error) {
	owners, err := s.graphBetaClient.ServicePrincipals().ID(id).Owners().Request().GetN(ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return owners, fmt.Errorf("failed to list owners for service principal: %w", err)
	}
	return owners, nil
}

func (s servicePrincipal) registerOwners(ctx context.Context, id azure.ServicePrincipalId, owners []msgraph.DirectoryObject) error {
	existing, err := s.getOwners(ctx, id)
	if err != nil {
		return err
	}
	newOwners := directoryobject.Difference(owners, directoryobject.MapToMsGraph(existing))

	for _, owner := range newOwners {
		body := directoryobject.ToOwnerPayload(owner)
		req := s.graphBetaClient.ServicePrincipals().ID(id).Owners().Request()
		err := req.JSONRequest(ctx, "POST", "/$ref", body, nil)
		if err != nil {
			return fmt.Errorf("failed to add owner '%s' to service principal: %w", *owner.ID, err)
		}
	}
	return nil
}

func (s servicePrincipal) revokeOwners(tx azure.Transaction, id azure.ServicePrincipalId) error {
	revoked, err := s.findRevokedOwners(tx, id)
	if err != nil {
		return err
	}
	if len(revoked) == 0 {
		return nil
	}
	for _, owner := range revoked {
		ownerId := *owner.ID
		req := s.graphBetaClient.ServicePrincipals().ID(id).Owners().ID(ownerId).Request()
		err := req.JSONRequest(tx.Ctx, "DELETE", "/$ref", nil, nil)
		if err != nil {
			return fmt.Errorf("failed to remove owner '%s' from service principal: %w", ownerId, err)
		}
	}
	return nil
}

func (s servicePrincipal) findRevokedOwners(tx azure.Transaction, id azure.ServicePrincipalId) ([]msgraphbeta.DirectoryObject, error) {
	revoked := make([]msgraphbeta.DirectoryObject, 0)

	desired, err := s.owners().get(tx)
	if err != nil {
		return revoked, err
	}

	existing, err := s.getOwners(tx.Ctx, id)
	if err != nil {
		return revoked, nil
	}

	existingMapped := directoryobject.MapToMsGraph(existing)
	difference := directoryobject.Difference(existingMapped, desired)
	revoked = directoryobject.MapToMsGraphBeta(difference)
	return revoked, nil
}
