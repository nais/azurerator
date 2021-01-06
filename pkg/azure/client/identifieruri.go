package client

import (
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

func (i identifierUri) set(tx azure.Transaction, application msgraph.Application) error {
	identifierUris := util.IdentifierUris(tx)
	app := util.EmptyApplication().IdentifierUriList(identifierUris).Build()
	if err := i.application.patch(tx.Ctx, *application.ID, app); err != nil {
		return fmt.Errorf("failed to add application identifier URI: %w", err)
	}
	return nil
}
