package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/directoryobject"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type applicationOwners struct {
	application
}

func (a application) owners() applicationOwners {
	return applicationOwners{a}
}

func (ao applicationOwners) get(ctx context.Context, id azure.ObjectId) ([]msgraph.DirectoryObject, error) {
	owners, err := ao.graphClient.Applications().ID(id).Owners().Request().GetN(ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return owners, fmt.Errorf("failed to list owners for application: %w", err)
	}
	return owners, nil
}

func (ao applicationOwners) register(ctx context.Context, id azure.ObjectId, owners []msgraph.DirectoryObject) error {
	existing, err := ao.get(ctx, id)
	if err != nil {
		return err
	}
	newOwners := directoryobject.Difference(owners, existing)

	for _, owner := range newOwners {
		body := directoryobject.ToOwnerPayload(owner)
		req := ao.graphClient.Applications().ID(id).Owners().Request()
		err := req.JSONRequest(ctx, "POST", "/$ref", body, nil)
		if err != nil {
			return fmt.Errorf("failed to add owner '%s' to application: %w", *owner.ID, err)
		}
	}
	return nil
}

func (ao applicationOwners) revoke(tx azure.Transaction, id azure.ObjectId) error {
	revoked, err := ao.findRevoked(tx, id)
	if err != nil {
		return err
	}
	if len(revoked) == 0 {
		return nil
	}
	for _, owner := range revoked {
		ownerId := *owner.ID
		req := ao.graphClient.Applications().ID(id).Owners().ID(ownerId).Request()
		err := req.JSONRequest(tx.Ctx, "DELETE", "/$ref", nil, nil)
		if err != nil {
			return fmt.Errorf("failed to remove owner '%s' from application: %w", ownerId, err)
		}
	}
	return nil
}

func (ao applicationOwners) findRevoked(tx azure.Transaction, id azure.ObjectId) ([]msgraph.DirectoryObject, error) {
	revoked := make([]msgraph.DirectoryObject, 0)
	desired, err := ao.teamowners().get(tx)
	if err != nil {
		return revoked, err
	}

	existing, err := ao.get(tx.Ctx, id)
	if err != nil {
		return revoked, nil
	}

	revoked = directoryobject.Difference(existing, desired)
	return revoked, nil
}
