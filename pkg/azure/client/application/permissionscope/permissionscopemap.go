package permissionscope

import (
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/permissions"
)

type Map map[string]msgraph.PermissionScope

func ToMap(scopes []msgraph.PermissionScope) Map {
	seen := make(Map)

	for _, scope := range scopes {
		seen.Add(scope)
	}

	return seen
}

func (m Map) Add(scope msgraph.PermissionScope) {
	name := *scope.Value

	if _, found := m[name]; !found {
		m[name] = scope
	}
}

func (m Map) ToSlice() []msgraph.PermissionScope {
	scopes := make([]msgraph.PermissionScope, 0)

	for _, scope := range m {
		scopes = append(scopes, scope)
	}

	return scopes
}

// ToCreate returns a Map describing the desired, non-existing scopes to be created.
func (m Map) ToCreate(desired permissions.Permissions) Map {
	toCreate := make(Map)

	// ensure default PermissionScope is created if it doesn't exist
	if _, found := m[permissions.DefaultPermissionScopeValue]; !found {
		toCreate[permissions.DefaultPermissionScopeValue] = DefaultScope()
	}

	for _, scope := range desired {
		if scope.Name == permissions.DefaultPermissionScopeValue {
			continue
		}

		if _, found := m[scope.Name]; !found {
			toCreate[scope.Name] = FromPermission(scope)
		}
	}

	return toCreate
}

// ToDisable returns a Map describing the existing, non-desired scopes to be disabled.
func (m Map) ToDisable(desired permissions.Permissions) Map {
	toDisable := make(Map)

	for _, scope := range m {
		name := *scope.Value
		if _, found := desired[name]; !found {
			disabledScope := scope
			disabledScope.IsEnabled = ptr.Bool(false)
			toDisable[name] = disabledScope
		}
	}

	// ensure default PermissionScope is not disabled
	delete(toDisable, permissions.DefaultPermissionScopeValue)
	return toDisable
}

// Unmodified returns a Map describing existing scopes that should not be modified.
// I.e. the difference of (existing - (toCreate + toDisable))
func (m Map) Unmodified(toCreate, toDisable Map) Map {
	unmodified := make(Map)

	for _, scope := range m {
		name := *scope.Value
		id := *scope.ID

		_, foundToCreate := toCreate[name]
		_, foundToDisable := toDisable[name]

		if foundToCreate || foundToDisable {
			continue
		}

		unmodified[name] = New(id, name)
	}

	return unmodified
}

func (m Map) ToPermissionList() permissions.PermissionList {
	result := make(permissions.PermissionList, 0)

	for _, scope := range m {
		result = append(result, permissions.FromPermissionScope(scope))
	}

	return result
}
