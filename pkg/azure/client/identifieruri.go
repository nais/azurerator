package client

import (
	"context"
	"fmt"

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
	if err := i.application.patch(ctx, *application.ID, app); err != nil {
		return fmt.Errorf("failed to add application identifier URI: %w", err)
	}
	return nil
}
