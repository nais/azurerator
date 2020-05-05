package client

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

const (
	// OAuth2 permission scope that the web API application exposes to client applications
	OAuth2DefaultAccessScope string = "defaultaccess"
)

func toPermissionScopes(id uuid.UUID) []msgraph.PermissionScope {
	defaultAccessScopeId := msgraph.UUID(id.String())
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
