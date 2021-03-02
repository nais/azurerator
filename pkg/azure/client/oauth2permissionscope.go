package client

import (
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

const (
	// OAuth2 permission scope that the web API application exposes to client applications
	OAuth2DefaultAccessScope string = "defaultaccess"
	// Apparently this just has to be unique per application, so we re-use this for all of our applications for consistency.
	OAuth2DefaultPermissionScopeId string = "00000000-1337-d34d-b33f-000000000000"
)

type oAuth2PermissionScopes struct {
	application
}

func (a application) oAuth2PermissionScopes() oAuth2PermissionScopes {
	return oAuth2PermissionScopes{a}
}

// ensure all other scopes than default scope are disabled
func (o oAuth2PermissionScopes) ensureValidScopes(tx azure.Transaction) error {
	existingScopes, err := o.getAll(tx)
	if err != nil {
		return err
	}

	for i, scope := range existingScopes {
		if *scope.ID == msgraph.UUID(OAuth2DefaultPermissionScopeId) {
			continue
		}
		scope.IsEnabled = ptr.Bool(false)
		existingScopes[i] = scope
	}

	return o.update(tx, existingScopes)
}

func (o oAuth2PermissionScopes) getAll(tx azure.Transaction) ([]msgraph.PermissionScope, error) {
	application, err := o.application.getByClientId(tx.Ctx, tx.Instance.GetClientId())
	if err != nil {
		return nil, fmt.Errorf("fetching application by client ID: %w", err)
	}
	return application.API.OAuth2PermissionScopes, nil
}

func (o oAuth2PermissionScopes) update(tx azure.Transaction, scopes []msgraph.PermissionScope) error {
	app := &msgraph.Application{API: &msgraph.APIApplication{OAuth2PermissionScopes: scopes}}
	if err := o.application.patch(tx.Ctx, tx.Instance.GetObjectId(), app); err != nil {
		return fmt.Errorf("patching application: %w", err)
	}
	return nil
}

func (o oAuth2PermissionScopes) defaultScopes() []msgraph.PermissionScope {
	defaultAccessScopeId := msgraph.UUID(OAuth2DefaultPermissionScopeId)
	return []msgraph.PermissionScope{
		{
			AdminConsentDescription: ptr.String(fmt.Sprintf("Gives adminconsent for scope %s", OAuth2DefaultAccessScope)),
			AdminConsentDisplayName: ptr.String(fmt.Sprintf("Adminconsent for scope %s", OAuth2DefaultAccessScope)),
			ID:                      &defaultAccessScopeId,
			IsEnabled:               ptr.Bool(true),
			Type:                    ptr.String("User"),
			Value:                   ptr.String(OAuth2DefaultAccessScope),
		},
	}
}
