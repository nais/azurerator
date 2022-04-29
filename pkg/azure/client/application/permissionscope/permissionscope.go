package permissionscope

import (
	"github.com/google/uuid"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/permissions"
)

func EnsureScopesRequireAdminConsent(scopes []msgraph.PermissionScope) []msgraph.PermissionScope {
	for i := range scopes {
		if *scopes[i].Type != permissions.DefaultScopeType {
			scopes[i].Type = ptr.String(permissions.DefaultScopeType)
		}
	}
	return scopes
}

func EnsureDefaultScopeIsEnabled(scopes []msgraph.PermissionScope) []msgraph.PermissionScope {
	for i := range scopes {
		if *scopes[i].Value == permissions.DefaultPermissionScopeValue && !*scopes[i].IsEnabled {
			scopes[i].IsEnabled = ptr.Bool(true)
		}
	}
	return scopes
}

func NewGenerateId(name string) msgraph.PermissionScope {
	id := msgraph.UUID(uuid.New().String())
	return New(id, name)
}

func New(id msgraph.UUID, name string) msgraph.PermissionScope {
	return msgraph.PermissionScope{
		AdminConsentDescription: ptr.String(name),
		AdminConsentDisplayName: ptr.String(name),
		ID:                      &id,
		IsEnabled:               ptr.Bool(true),
		Type:                    ptr.String(permissions.DefaultScopeType),
		Value:                   ptr.String(name),
	}
}

func DefaultScope() msgraph.PermissionScope {
	id := msgraph.UUID(permissions.DefaultPermissionScopeId)
	return New(id, permissions.DefaultPermissionScopeValue)
}

func FromPermission(permission permissions.Permission) msgraph.PermissionScope {
	return New(permission.ID, permission.Name)
}

func RemoveDisabled(application msgraph.Application) []msgraph.PermissionScope {
	desired := make([]msgraph.PermissionScope, 0)

	if application.API == nil {
		return desired
	}

	for _, scope := range application.API.OAuth2PermissionScopes {
		if *scope.IsEnabled {
			desired = append(desired, scope)
		}
	}

	return desired
}
