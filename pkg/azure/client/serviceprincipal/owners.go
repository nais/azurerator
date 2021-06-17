package serviceprincipal

import (
	"fmt"

	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	directoryobject2 "github.com/nais/azureator/pkg/azure/directoryobject"
	"github.com/nais/azureator/pkg/azure/transaction"
)

type owners struct {
	azure.RuntimeClient
}

func newOwners(client azure.RuntimeClient) azure.ServicePrincipalOwners {
	return &owners{RuntimeClient: client}
}

func (o owners) Process(tx transaction.Transaction, desired []msgraph.DirectoryObject) error {
	existing, err := o.get(tx)
	if err != nil {
		return err
	}

	newOwners := directoryobject2.Difference(desired, existing)

	if err := o.registerFor(tx, newOwners); err != nil {
		return fmt.Errorf("registering new owners for service principal: %w", err)
	}

	revokedOwners := o.revoked(desired, existing)
	if err := o.revokeFor(tx, revokedOwners); err != nil {
		return fmt.Errorf("revoking owners for service principal: %w", err)
	}

	return nil
}

func (o owners) get(tx transaction.Transaction) ([]msgraph.DirectoryObject, error) {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()
	owners, err := o.GraphClient().ServicePrincipals().ID(servicePrincipalId).Owners().Request().GetN(tx.Ctx, o.MaxNumberOfPagesToFetch())
	if err != nil {
		return owners, fmt.Errorf("listing owners for service principal: %w", err)
	}
	return owners, nil
}

func (o owners) registerFor(tx transaction.Transaction, owners []msgraph.DirectoryObject) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	for _, owner := range owners {
		body := directoryobject2.ToOwnerPayload(owner)
		req := o.GraphClient().ServicePrincipals().ID(servicePrincipalId).Owners().Request()
		err := req.JSONRequest(tx.Ctx, "POST", "/$ref", body, nil)
		if err != nil {
			return fmt.Errorf("adding owner '%s' to service principal: %w", *owner.ID, err)
		}
	}
	return nil
}

func (o owners) revokeFor(tx transaction.Transaction, revoked []msgraph.DirectoryObject) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	for _, owner := range revoked {
		ownerId := *owner.ID
		req := o.GraphClient().ServicePrincipals().ID(servicePrincipalId).Owners().ID(ownerId).Request()
		err := req.JSONRequest(tx.Ctx, "DELETE", "/$ref", nil, nil)
		if err != nil {
			return fmt.Errorf("removing owner '%s' from service principal: %w", ownerId, err)
		}
	}
	return nil
}

func (o owners) revoked(desired []msgraph.DirectoryObject, existing []msgraph.DirectoryObject) []msgraph.DirectoryObject {
	revoked := directoryobject2.Difference(existing, desired)
	return revoked
}
