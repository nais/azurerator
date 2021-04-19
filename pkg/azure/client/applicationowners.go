package client

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/directoryobject"
	msgraph "github.com/nais/msgraph.go/v1.0"
)

type applicationOwners struct {
	application
}

func (a application) owners() applicationOwners {
	return applicationOwners{a}
}

func (ao applicationOwners) get(tx azure.Transaction) ([]msgraph.DirectoryObject, error) {
	objectId := tx.Instance.GetObjectId()

	owners, err := ao.graphClient.Applications().ID(objectId).Owners().Request().GetN(tx.Ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return owners, fmt.Errorf("listing owners for application: %w", err)
	}
	return owners, nil
}

func (ao applicationOwners) process(tx azure.Transaction, desired []msgraph.DirectoryObject) error {
	existing, err := ao.get(tx)
	if err != nil {
		return err
	}

	newOwners := directoryobject.Difference(desired, existing)
	if err := ao.registerFor(tx, newOwners); err != nil {
		return fmt.Errorf("registering owners for application: %w", err)
	}

	revoked := directoryobject.Difference(existing, desired)
	if err := ao.revokeFor(tx, revoked); err != nil {
		return fmt.Errorf("revoking owners for application: %w", err)
	}

	return nil
}

func (ao applicationOwners) registerFor(tx azure.Transaction, owners []msgraph.DirectoryObject) error {
	objectId := tx.Instance.GetObjectId()

	for _, owner := range owners {
		body := directoryobject.ToOwnerPayload(owner)
		req := ao.graphClient.Applications().ID(objectId).Owners().Request()
		err := req.JSONRequest(tx.Ctx, "POST", "/$ref", body, nil)
		if err != nil {
			return fmt.Errorf("adding owner '%s' to application: %w", *owner.ID, err)
		}
	}
	return nil
}

func (ao applicationOwners) revokeFor(tx azure.Transaction, revoked []msgraph.DirectoryObject) error {
	objectId := tx.Instance.GetObjectId()

	for _, owner := range revoked {
		ownerId := *owner.ID
		req := ao.graphClient.Applications().ID(objectId).Owners().ID(ownerId).Request()

		err := req.JSONRequest(tx.Ctx, "DELETE", "/$ref", nil, nil)
		if err != nil {
			return fmt.Errorf("removing owner '%s' from application: %w", ownerId, err)
		}
	}
	return nil
}
