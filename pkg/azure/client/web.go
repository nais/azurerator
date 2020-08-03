package client

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type web struct {
	application
}

func (a application) web() web {
	return web{a}
}

func (w web) app(tx azure.Transaction) *msgraph.WebApplication {
	return &msgraph.WebApplication{
		LogoutURL:    ptr.String(tx.Instance.Spec.LogoutUrl),
		RedirectUris: util.GetReplyUrlsStringSlice(tx.Instance),
		ImplicitGrantSettings: &msgraph.ImplicitGrantSettings{
			EnableIDTokenIssuance:     ptr.Bool(false),
			EnableAccessTokenIssuance: ptr.Bool(false),
		},
	}
}

func (w web) update(tx azure.Transaction) error {
	objectId := tx.Instance.Status.ObjectId
	webApp := w.app(tx)
	app := util.EmptyApplication().Web(webApp).Build()
	if err := w.application.update(tx.Ctx, objectId, app); err != nil {
		return fmt.Errorf("failed to update web application: %w", err)
	}
	return nil
}
