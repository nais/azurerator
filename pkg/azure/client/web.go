package client

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Workaround to include empty array of RedirectUris in JSON serialization.
// The autogenerated library code uses 'omitempty' for RedirectUris, which when empty
// leaves the list of redirect URIs unchanged and non-empty.
type webApi struct {
	RedirectUris          []string                       `json:"redirectUris"`
	LogoutURL             *string                        `json:"logoutUrl,omitempty"`
	ImplicitGrantSettings *msgraph.ImplicitGrantSettings `json:"implicitGrantSettings,omitempty"`
}

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
	web := webApi{
		LogoutURL:    ptr.String(tx.Instance.Spec.LogoutUrl),
		RedirectUris: util.GetReplyUrlsStringSlice(tx.Instance),
		ImplicitGrantSettings: &msgraph.ImplicitGrantSettings{
			EnableIDTokenIssuance:     ptr.Bool(false),
			EnableAccessTokenIssuance: ptr.Bool(false),
		},
	}
	app := &struct {
		msgraph.DirectoryObject
		Web webApi `json:"web"`
	}{
		Web: web,
	}
	appReq := w.graphClient.Applications().ID(objectId).Request()
	if err := appReq.JSONRequest(tx.Ctx, "PATCH", "", app, nil); err != nil {
		return fmt.Errorf("failed to update web application: %w", err)
	}
	return nil
}
