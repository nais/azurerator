package approle

import (
	"github.com/google/uuid"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/permissions"
)

func New(id msgraph.UUID, name string) msgraph.AppRole {
	return msgraph.AppRole{
		AllowedMemberTypes: []string{"Application"},
		Description:        ptr.String(name),
		DisplayName:        ptr.String(name),
		ID:                 &id,
		IsEnabled:          ptr.Bool(true),
		Value:              ptr.String(name),
	}
}

func NewGenerateId(name string) msgraph.AppRole {
	id := msgraph.UUID(uuid.New().String())
	return New(id, name)
}

func DefaultRole() msgraph.AppRole {
	return New(msgraph.UUID(permissions.DefaultAppRoleId), permissions.DefaultAppRoleValue)
}

func DefaultGroupRole() msgraph.AppRole {
	return New(msgraph.UUID(permissions.DefaultGroupRoleId), permissions.DefaultGroupRoleValue)
}

func EnsureDefaultAppRoleIsEnabled(scopes []msgraph.AppRole) []msgraph.AppRole {
	for i := range scopes {
		if *scopes[i].Value == permissions.DefaultAppRoleValue && !*scopes[i].IsEnabled {
			scopes[i].IsEnabled = ptr.Bool(true)
		}
	}
	return scopes
}

func FromPermission(permission permissions.Permission) msgraph.AppRole {
	return New(permission.ID, permission.Name)
}

func RemoveDisabled(application msgraph.Application) []msgraph.AppRole {
	desired := make([]msgraph.AppRole, 0)

	for _, role := range application.AppRoles {
		if *role.IsEnabled {
			desired = append(desired, role)
		}
	}

	return desired
}
