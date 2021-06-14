package permissions

import (
	"github.com/google/uuid"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"
)

type Permissions map[string]Permission

func (p Permissions) Add(permission Permission) {
	name := permission.Name

	if _, found := p[name]; !found {
		p[name] = permission
	}
}

// Permission is a struct defining common fields used for generating and managing both AppRole and PermissionScope.
type Permission struct {
	Name    string
	ID      msgraph.UUID
	Enabled bool
}

func New(id msgraph.UUID, name string, enabled bool) Permission {
	return Permission{
		Name:    name,
		ID:      id,
		Enabled: enabled,
	}
}

func FromAppRole(in msgraph.AppRole) Permission {
	return New(*in.ID, *in.Value, *in.IsEnabled)
}

func FromPermissionScope(in msgraph.PermissionScope) Permission {
	return New(*in.ID, *in.Value, *in.IsEnabled)
}

func NewGenerateIdEnabled(name string) Permission {
	return newGenerateId(name, true)
}

func NewGenerateIdDisabled(name string) Permission {
	return newGenerateId(name, false)
}

func newGenerateId(name string, enabled bool) Permission {
	id := msgraph.UUID(uuid.New().String())
	return New(id, name, enabled)
}

// GenerateDesiredPermissionSet extracts the desired set of permissions from the given nais_io_v1.AzureAdApplication.
// It generates UUIDs for each permission to be used when registering the permission to Azure AD.
// This is to ensure that PermissionScopes and AppRoles created using these Permissions as basis have the same values for a number of fields.
// See https://stackoverflow.com/a/59550249/11868133 for details on this limitation.
func GenerateDesiredPermissionSet(in naisiov1.AzureAdApplication) Permissions {
	permissions := make(Permissions)

	desiredRoles := flattenRoles(in)
	desiredScopes := flattenScopes(in)

	for _, permission := range append(desiredScopes, desiredRoles...) {
		permissions.Add(NewGenerateIdEnabled(permission))
	}

	return permissions
}

// GenerateDesiredPermissionSetPreserveExisting extracts the desired set of permissions from the given nais_io_v1.AzureAdApplication,
// It generates UUIDs for each non-existing permission to be used when registering the permission to Azure AD.
// This is to ensure that PermissionScopes and AppRoles created using these Permissions as basis have the same values for a number of fields.
// See https://stackoverflow.com/a/59550249/11868133 for details on this limitation.
// Existing permissions (and their IDs) are preserved.
func GenerateDesiredPermissionSetPreserveExisting(in naisiov1.AzureAdApplication, existing msgraph.Application) Permissions {
	desired := GenerateDesiredPermissionSet(in)
	actual := ExtractPermissions(&existing)

	for key := range desired {
		if actualValue, found := actual[key]; found {
			desired[key] = actualValue
		}
	}

	return desired
}

// ExtractPermissions extracts the actual permissions as they're defined in the msgraph.Application resource in Azure AD.
// Permissions (whether they're a PermissionScope or AppRole) in Azure AD with the same value/name must have the same ID.
func ExtractPermissions(app *msgraph.Application) Permissions {
	permissions := make(Permissions)

	for _, scope := range app.API.OAuth2PermissionScopes {
		permissions.Add(FromPermissionScope(scope))
	}

	for _, role := range app.AppRoles {
		permissions.Add(FromAppRole(role))
	}

	return permissions
}

func flattenScopes(in naisiov1.AzureAdApplication) []string {
	return flatten(in.Spec.PreAuthorizedApplications, func(rule naisiov1.AccessPolicyRule) []string {
		if rule.Permissions != nil && len(rule.Permissions.Scopes) > 0 {
			return rule.Permissions.Scopes
		} else {
			return make([]string, 0)
		}
	})
}

func flattenRoles(in naisiov1.AzureAdApplication) []string {
	return flatten(in.Spec.PreAuthorizedApplications, func(rule naisiov1.AccessPolicyRule) []string {
		if rule.Permissions != nil && len(rule.Permissions.Roles) > 0 {
			return rule.Permissions.Roles
		} else {
			return make([]string, 0)
		}
	})
}

func flatten(in []naisiov1.AccessPolicyRule, rule func(rule naisiov1.AccessPolicyRule) []string) []string {
	result := make([]string, 0)

	for _, app := range in {
		for _, permission := range rule(app) {
			result = append(result, permission)
		}
	}

	return result
}
