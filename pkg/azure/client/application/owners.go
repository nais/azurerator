package application

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

func newOwners(client azure.RuntimeClient) azure.ApplicationOwners {
	return owners{RuntimeClient: client}
}

func (o owners) Process(tx transaction.Transaction, desired []msgraph.DirectoryObject) error {
	existing, err := o.get(tx)
	if err != nil {
		return err
	}

	newOwners := directoryobject2.Difference(desired, existing)
	if err := o.registerFor(tx, newOwners); err != nil {
		return fmt.Errorf("registering owners for application: %w", err)
	}

	revoked := directoryobject2.Difference(existing, desired)
	if err := o.revokeFor(tx, revoked); err != nil {
		return fmt.Errorf("revoking owners for application: %w", err)
	}

	return nil
}

func (o owners) get(tx transaction.Transaction) ([]msgraph.DirectoryObject, error) {
	objectId := tx.Instance.GetObjectId()

	owners, err := o.GraphClient().Applications().ID(objectId).Owners().Request().GetN(tx.Ctx, o.MaxNumberOfPagesToFetch())
	if err != nil {
		return owners, fmt.Errorf("listing owners for application: %w", err)
	}
	return owners, nil
}

func (o owners) registerFor(tx transaction.Transaction, owners []msgraph.DirectoryObject) error {
	objectId := tx.Instance.GetObjectId()

	for _, owner := range owners {
		body := directoryobject2.ToOwnerPayload(owner)
		req := o.GraphClient().Applications().ID(objectId).Owners().Request()
		err := req.JSONRequest(tx.Ctx, "POST", "/$ref", body, nil)
		if err != nil {
			return fmt.Errorf("adding owner '%s' to application: %w", *owner.ID, err)
		}
	}
	return nil
}

func (o owners) revokeFor(tx transaction.Transaction, revoked []msgraph.DirectoryObject) error {
	objectId := tx.Instance.GetObjectId()

	for _, owner := range revoked {
		ownerId := *owner.ID
		req := o.GraphClient().Applications().ID(objectId).Owners().ID(ownerId).Request()

		err := req.JSONRequest(tx.Ctx, "DELETE", "/$ref", nil, nil)
		if err != nil {
			return fmt.Errorf("removing owner '%s' from application: %w", ownerId, err)
		}
	}
	return nil
}
