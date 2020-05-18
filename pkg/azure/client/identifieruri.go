package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type identifierUri struct {
	application
}

func (a application) identifierUri() identifierUri {
	return identifierUri{a}
}

func (i identifierUri) set(ctx context.Context, application msgraph.Application) error {
	identifierUri := util.IdentifierUri(*application.AppID)
	app := util.EmptyApplication().IdentifierUri(identifierUri).Build()
	if err := i.application.update(ctx, *application.ID, app); err != nil {
		return fmt.Errorf("failed to add application identifier URI: %w", err)
	}
	return nil
}

func (i identifierUri) update(tx azure.Transaction) error {
	clientId := tx.Instance.Status.ClientId
	objectId := tx.Instance.Status.ObjectId
	uri := util.IdentifierUri(clientId)
	app := util.EmptyApplication().IdentifierUri(uri).Build()
	if err := i.application.update(tx.Ctx, objectId, app); err != nil {
		return fmt.Errorf("failed to update application identifier URI: %w", err)
	}
	return nil
}
