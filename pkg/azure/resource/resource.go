package resource

import (
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/client/application/approle"
	"github.com/nais/azureator/pkg/azure/client/application/permissionscope"
	"github.com/nais/azureator/pkg/azure/permissions"
)

// Resource contains metadata that identifies a resource (e.g. User, Groups, Application, or Service Principal) within Azure AD.
type Resource struct {
	Name                      string        `json:"name"`
	ClientId                  string        `json:"clientId"`
	ObjectId                  string        `json:"-"`
	PrincipalType             PrincipalType `json:"-"`
	naisiov1.AccessPolicyRule `json:"-"`
}

func (r Resource) ToPreAuthorizedApp(permissions permissions.Permissions) msgraph.PreAuthorizedApplication {
	clientId := r.ClientId

	desiredPermissions := []string{
		permissionscope.DefaultAccessScopeValue,
	}

	if r.AccessPolicyRule.Permissions != nil {
		for _, scope := range r.AccessPolicyRule.Permissions.Scopes {
			desiredPermissions = append(desiredPermissions, string(scope))
		}
	}

	permissionIDs := permissions.
		Filter(desiredPermissions...).
		PermissionIDs()

	return msgraph.PreAuthorizedApplication{
		AppID:                  &clientId,
		DelegatedPermissionIDs: permissionIDs,
	}
}

func (r Resource) ToAppRoleAssignment(target string, permission permissions.Permission) msgraph.AppRoleAssignment {
	return msgraph.AppRoleAssignment{
		AppRoleID:            &permission.ID,                          // The ID of the AppRole belonging to the target resource to be assigned
		PrincipalDisplayName: ptr.String(r.Name),                      // Name of the assignee
		PrincipalID:          (*msgraph.UUID)(ptr.String(r.ObjectId)), // Service Principal ID for the assignee, i.e. the principal that should be assigned to the app role
		PrincipalType:        ptr.String(string(r.PrincipalType)),     // The Principal type of the assignee, e.g. ServicePrincipal or Group
		ResourceID:           (*msgraph.UUID)(ptr.String(target)),     // Service Principal ID for the target resource, i.e. the application/service principal that owns the app role
	}
}

type Resources []Resource

func (r Resources) FilterByRole(role permissions.Permission) Resources {
	result := make(Resources, 0)

	for _, re := range r {
		seen := make(map[naisiov1.AccessPolicyPermission]bool)

		if re.Permissions == nil {
			continue
		}

		for _, desiredRole := range re.Permissions.Roles {
			if string(desiredRole) == role.Name && !seen[desiredRole] {
				seen[desiredRole] = true
				result = append(result, re)
			}
		}
	}

	return result
}

func (r Resources) FilterByPrincipalType(principalType PrincipalType) Resources {
	result := make(Resources, 0)

	for _, re := range r {
		if re.PrincipalType == principalType {
			result = append(result, re)
		}
	}

	return result
}

func (r Resources) ExtractDesiredAssignees(principalType PrincipalType, role permissions.Permission) Resources {
	switch principalType {
	case PrincipalTypeGroup:
		// ensure that default group role is assigned to all Groups
		if role.ID == msgraph.UUID(approle.DefaultGroupRoleId) {
			return r
		}
	case PrincipalTypeServicePrincipal:
		// ensure that default app role is assigned to all ServicePrincipals
		if role.Name == approle.DefaultAppRoleValue {
			return r
		}
	}

	return r.FilterByRole(role)
}

type PrincipalType string

const (
	PrincipalTypeGroup            PrincipalType = "Group"
	PrincipalTypeServicePrincipal PrincipalType = "ServicePrincipal"
	PrincipalTypeUser             PrincipalType = "User"
)
