package application

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
)

type identifierUri struct {
	azure.Application
}

func newIdentifierUri(application azure.Application) azure.IdentifierUri {
	return identifierUri{Application: application}
}

func (i identifierUri) Set(tx azure.Transaction) error {
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
