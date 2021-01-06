package client

import (
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

const (
	DefaultAppRole   string = "access_as_application"
	DefaultAppRoleId string = "00000001-abcd-9001-0000-000000000000"
)

type appRoles struct {
	application
}

func (a application) appRoles() appRoles {
	return appRoles{a}
}

func (a appRoles) ensureExists(tx azure.Transaction, role msgraph.AppRole) ([]msgraph.AppRole, error) {
	roles, err := a.getAll(tx)
	if err != nil {
		return nil, fmt.Errorf("fetching approles for application: %w", err)
	}

	roles, err = a.disableConflictingRoles(tx, role, roles)
	if err != nil {
		return nil, fmt.Errorf("disabling duplicate roles: %w", err)
	}

	roles = filterDisabledRoles(roles)

	if exists := roleExistsInRoles(role, roles); !exists {
		roles = append(roles, role)
	}

	return roles, nil
}

func (a appRoles) getAll(tx azure.Transaction) ([]msgraph.AppRole, error) {
	application, err := a.application.getByClientId(tx.Ctx, tx.Instance.Status.ClientId)
	if err != nil {
		return nil, fmt.Errorf("fetching application by client ID: %w", err)
	}
	return application.AppRoles, nil
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

// Disable roles with other IDs that have the same Value (i.e. the emitted string value in the _roles_ claim - must be unique per Application)
func (a appRoles) disableConflictingRoles(tx azure.Transaction, role msgraph.AppRole, roles []msgraph.AppRole) ([]msgraph.AppRole, error) {
	result := make([]msgraph.AppRole, 0)
	for _, r := range roles {
		if *role.ID != *r.ID && *role.Value == *r.Value {
			tx.Log.Debugf("disabling role '%s' with duplicate value '%s'", *r.ID, *r.Value)
			r.IsEnabled = ptr.Bool(false)
		}
		result = append(result, r)
	}

	if err := a.update(tx, result); err != nil {
		return nil, fmt.Errorf("updating roles for application: %w", err)
	}

	return result, nil
}

func (a appRoles) update(tx azure.Transaction, roles []msgraph.AppRole) error {
	app := util.EmptyApplication().AppRoles(roles).Build()
	if err := a.application.patch(tx.Ctx, tx.Instance.Status.ObjectId, app); err != nil {
		return fmt.Errorf("patching application: %w", err)
	}
	return nil
}

func roleExistsInRoles(role msgraph.AppRole, roles []msgraph.AppRole) bool {
	for _, r := range roles {
		if *role.ID == *r.ID && *role.Value == *r.Value {
			return true
		}
	}
	return false
}

func filterDisabledRoles(roles []msgraph.AppRole) []msgraph.AppRole {
	filtered := make([]msgraph.AppRole, 0)
	for _, role := range roles {
		if *role.IsEnabled {
			filtered = append(filtered, role)
		}
	}
	return filtered
}
