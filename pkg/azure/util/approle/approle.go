package approle

import (
	"github.com/google/uuid"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/util/permissions"
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
	return New(msgraph.UUID(DefaultAppRoleId), DefaultAppRoleValue)
}

func EnsureDefaultAppRoleIsEnabled(scopes []msgraph.AppRole) []msgraph.AppRole {
	for i := range scopes {
		if *scopes[i].Value == DefaultAppRoleValue && !*scopes[i].IsEnabled {
			scopes[i].IsEnabled = ptr.Bool(true)
		}
	}
	return scopes
}

func FromPermission(permission permissions.Permission) msgraph.AppRole {
	return New(permission.ID, permission.Name)
}
