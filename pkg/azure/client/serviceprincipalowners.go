package client

import (
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

func (so servicePrincipalOwners) get(tx azure.Transaction) ([]msgraphbeta.DirectoryObject, error) {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()
	owners, err := so.graphBetaClient.ServicePrincipals().ID(servicePrincipalId).Owners().Request().GetN(tx.Ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return owners, fmt.Errorf("listing owners for service principal: %w", err)
	}
	return owners, nil
}

func (so servicePrincipalOwners) process(tx azure.Transaction, desired []msgraph.DirectoryObject) error {
	existing, err := so.get(tx)
	if err != nil {
		return err
	}

	newOwners := directoryobject.Difference(desired, directoryobject.MapToMsGraph(existing))

	if err := so.registerFor(tx, newOwners); err != nil {
		return fmt.Errorf("registering new owners for service principal: %w", err)
	}

	revokedOwners := so.revoked(desired, existing)
	if err := so.revokeFor(tx, revokedOwners); err != nil {
		return fmt.Errorf("revoking owners for service principal: %w", err)
	}

	return nil
}

func (so servicePrincipalOwners) registerFor(tx azure.Transaction, owners []msgraph.DirectoryObject) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	for _, owner := range owners {
		body := directoryobject.ToOwnerPayload(owner)
		req := so.graphBetaClient.ServicePrincipals().ID(servicePrincipalId).Owners().Request()
		err := req.JSONRequest(tx.Ctx, "POST", "/$ref", body, nil)
		if err != nil {
			return fmt.Errorf("adding owner '%s' to service principal: %w", *owner.ID, err)
		}
	}
	return nil
}

func (so servicePrincipalOwners) revokeFor(tx azure.Transaction, revoked []msgraphbeta.DirectoryObject) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	for _, owner := range revoked {
		ownerId := *owner.ID
		req := so.graphBetaClient.ServicePrincipals().ID(servicePrincipalId).Owners().ID(ownerId).Request()
		err := req.JSONRequest(tx.Ctx, "DELETE", "/$ref", nil, nil)
		if err != nil {
			return fmt.Errorf("removing owner '%s' from service principal: %w", ownerId, err)
		}
	}
	return nil
}

func (so servicePrincipalOwners) revoked(desired []msgraph.DirectoryObject, existing []msgraphbeta.DirectoryObject) []msgraphbeta.DirectoryObject {
	existingMapped := directoryobject.MapToMsGraph(existing)
	difference := directoryobject.Difference(existingMapped, desired)
	revoked := directoryobject.MapToMsGraphBeta(difference)
	return revoked
}
