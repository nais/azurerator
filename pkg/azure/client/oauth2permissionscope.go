package client

import (
	"fmt"

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
	client
}

func (c client) oAuth2PermissionScopes() oAuth2PermissionScopes {
	return oAuth2PermissionScopes{c}
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
