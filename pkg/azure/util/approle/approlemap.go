package approle

import (
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/util/permissions"
)

type Map map[string]msgraph.AppRole

func ToMap(roles []msgraph.AppRole) Map {
	seen := make(Map)

	for _, role := range roles {
		seen.Add(role)
	}

	return seen
}

func (m Map) Add(role msgraph.AppRole) {
	name := *role.Value

	if _, found := m[name]; !found {
		m[name] = role
	}
}

func (m Map) ToSlice() []msgraph.AppRole {
	roles := make([]msgraph.AppRole, 0)

	for _, appRole := range m {
		roles = append(roles, appRole)
	}

	return roles
}

// ToCreate returns a Map describing the desired, non-existing roles to be created.
func (m Map) ToCreate(desired permissions.Permissions) Map {
	toCreate := make(Map)

	// ensure default AppRole is created if it doesn't exist
	if _, found := m[DefaultAppRoleValue]; !found {
		toCreate[DefaultAppRoleValue] = DefaultRole()
	}

	for _, role := range desired {
		if role.Name == DefaultAppRoleValue {
			continue
		}

		if _, found := m[role.Name]; !found {
			toCreate[role.Name] = FromPermission(role)
		}
	}

	return toCreate
}

// ToDisable returns a Map describing the existing, non-desired scopes to be disabled.
func (m Map) ToDisable(desired permissions.Permissions) Map {
	toDisable := make(Map)

	for _, role := range m {
		name := *role.Value
		if _, found := desired[name]; !found {
			disabledRole := role
			disabledRole.IsEnabled = ptr.Bool(false)
			toDisable[name] = disabledRole
		}
	}

	// ensure default AppRole is not disabled
	if _, found := toDisable[DefaultAppRoleValue]; found {
		delete(toDisable, DefaultAppRoleValue)
	}

	return toDisable
}

// Unmodified returns a Map describing existing scopes that should not be modified.
// I.e. the difference of (existing - (toCreate + toDisable))
func (m Map) Unmodified(toCreate, toDisable Map) Map {
	unmodified := make(Map)

	for _, role := range m {
		name := *role.Value
		id := *role.ID

		_, foundToCreate := toCreate[name]
		_, foundToDisable := toDisable[name]

		if foundToCreate || foundToDisable {
			continue
		}

		unmodified[name] = New(id, name)
	}

	return unmodified
}
