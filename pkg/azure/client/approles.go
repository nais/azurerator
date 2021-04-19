package client

import (
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
)

const (
	DefaultAppRole     string = "access_as_application"
	DefaultAppRoleId   string = "00000001-abcd-9001-0000-000000000000"
	DefaultGroupRoleId string = "00000000-0000-0000-0000-000000000000"
)

type appRoles struct {
	application
}

func (a application) appRoles() appRoles {
	return appRoles{a}
}

func (a appRoles) getAll(tx azure.Transaction) ([]msgraph.AppRole, error) {
	application, err := a.application.getByClientId(tx.Ctx, tx.Instance.GetClientId())
	if err != nil {
		return nil, fmt.Errorf("fetching application by client ID: %w", err)
	}
	return application.AppRoles, nil
}

func (a appRoles) getOrGenerateRoleID(tx azure.Transaction, role msgraph.AppRole) (*msgraph.UUID, error) {
	roles, err := a.getAll(tx)
	if err != nil {
		return nil, fmt.Errorf("fetching approles for application: %w", err)
	}

	if role, found := roleExistsInRoles(role, roles); found {
		return role.ID, nil
	}

	roles = append(roles, role)
	if err := a.update(tx, roles); err != nil {
		return nil, fmt.Errorf("adding approle '%s' (%s) for application: %w", *role.Value, *role.ID, err)
	}

	return role.ID, nil
}

func (a appRoles) defaultRole() msgraph.AppRole {
	roleId := msgraph.UUID(DefaultAppRoleId)
	allowedMemberTypes := []string{"Application"}
	return msgraph.AppRole{
		Object:             msgraph.Object{},
		AllowedMemberTypes: allowedMemberTypes,
		Description:        ptr.String(DefaultAppRole),
		DisplayName:        ptr.String(DefaultAppRole),
		ID:                 &roleId,
		IsEnabled:          ptr.Bool(true),
		Value:              ptr.String(DefaultAppRole),
	}
}

func (a appRoles) update(tx azure.Transaction, roles []msgraph.AppRole) error {
	app := util.EmptyApplication().AppRoles(roles).Build()
	if err := a.application.patch(tx.Ctx, tx.Instance.GetObjectId(), app); err != nil {
		return fmt.Errorf("patching application: %w", err)
	}
	return nil
}

func roleExistsInRoles(role msgraph.AppRole, roles []msgraph.AppRole) (*msgraph.AppRole, bool) {
	for _, r := range roles {
		if *role.Value == *r.Value {
			return &r, true
		}
	}
	return nil, false
}
