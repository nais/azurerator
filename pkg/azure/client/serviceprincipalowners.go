package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/directoryobject"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type servicePrincipalOwners struct {
	servicePrincipal
}

func (s servicePrincipal) owners() servicePrincipalOwners {
	return servicePrincipalOwners{s}
}

func (so servicePrincipalOwners) get(ctx context.Context, id azure.ServicePrincipalId) ([]msgraphbeta.DirectoryObject, error) {
	owners, err := so.graphBetaClient.ServicePrincipals().ID(id).Owners().Request().GetN(ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return owners, fmt.Errorf("failed to list owners for service principal: %w", err)
	}
	return owners, nil
}

func (so servicePrincipalOwners) register(ctx context.Context, id azure.ServicePrincipalId, owners []msgraph.DirectoryObject) error {
	existing, err := so.get(ctx, id)
	if err != nil {
		return err
	}
	newOwners := directoryobject.Difference(owners, directoryobject.MapToMsGraph(existing))

	for _, owner := range newOwners {
		body := directoryobject.ToOwnerPayload(owner)
		req := so.graphBetaClient.ServicePrincipals().ID(id).Owners().Request()
		err := req.JSONRequest(ctx, "POST", "/$ref", body, nil)
		if err != nil {
			return fmt.Errorf("failed to add owner '%s' to service principal: %w", *owner.ID, err)
		}
	}
	return nil
}

func (so servicePrincipalOwners) revoke(tx azure.Transaction, id azure.ServicePrincipalId) error {
	revoked, err := so.findRevoked(tx, id)
	if err != nil {
		return err
	}
	if len(revoked) == 0 {
		return nil
	}
	for _, owner := range revoked {
		ownerId := *owner.ID
		req := so.graphBetaClient.ServicePrincipals().ID(id).Owners().ID(ownerId).Request()
		err := req.JSONRequest(tx.Ctx, "DELETE", "/$ref", nil, nil)
		if err != nil {
			return fmt.Errorf("failed to remove owner '%s' from service principal: %w", ownerId, err)
		}
	}
	return nil
}

func (so servicePrincipalOwners) findRevoked(tx azure.Transaction, id azure.ServicePrincipalId) ([]msgraphbeta.DirectoryObject, error) {
	revoked := make([]msgraphbeta.DirectoryObject, 0)

	desired, err := so.teamowners().get(tx)
	if err != nil {
		return revoked, err
	}

	existing, err := so.get(tx.Ctx, id)
	if err != nil {
		return revoked, nil
	}

	existingMapped := directoryobject.MapToMsGraph(existing)
	difference := directoryobject.Difference(existingMapped, desired)
	revoked = directoryobject.MapToMsGraphBeta(difference)
	return revoked, nil
}
