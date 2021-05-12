package client

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
)

type identifierUri struct {
	application
}

func (a application) identifierUri() identifierUri {
	return identifierUri{a}
}

func (i identifierUri) set(tx azure.Transaction) error {
	objectId := tx.Instance.GetObjectId()
	identifierUris := util.IdentifierUris(tx)
	app := util.EmptyApplication().IdentifierUriList(identifierUris).Build()
	if err := i.application.patch(tx.Ctx, objectId, app); err != nil {
		return fmt.Errorf("failed to add application identifier URI: %w", err)
	}
	return nil
}
