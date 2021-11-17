package identifieruri

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/transaction"
	"github.com/nais/azureator/pkg/azure/util"
)

type IdentifierUri interface {
	Set(tx transaction.Transaction) error
}

type identifierUri struct {
	Application
}

type Application interface {
	Patch(ctx context.Context, id azure.ObjectId, application interface{}) error
}

func NewIdentifierUri(application Application) IdentifierUri {
	return identifierUri{Application: application}
}

func (i identifierUri) Set(tx transaction.Transaction) error {
	objectId := tx.Instance.GetObjectId()
	identifierUris := util.IdentifierUris(tx)
	app := util.EmptyApplication().
		IdentifierUriList(identifierUris).
		Build()
	if err := i.Application.Patch(tx.Ctx, objectId, app); err != nil {
		return fmt.Errorf("failed to add application identifier URI: %w", err)
	}
	return nil
}
