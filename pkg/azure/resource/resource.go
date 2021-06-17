package resource

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/client/application/permissionscope"
	"github.com/nais/azureator/pkg/azure/permissions"
)

// Resource contains metadata that identifies a resource (e.g. User, Groups, Application, or Service Principal) within Azure AD.
type Resource struct {
	Name                        string        `json:"name"`
	ClientId                    string        `json:"clientId"`
	ObjectId                    string        `json:"-"`
	PrincipalType               PrincipalType `json:"-"`
	nais_io_v1.AccessPolicyRule `json:"-"`
}

func (r Resource) ToPreAuthorizedApp(permissions permissions.Permissions) msgraph.PreAuthorizedApplication {
	clientId := r.ClientId

	desiredPermissions := []string{
		permissionscope.DefaultAccessScopeValue,
	}

	if r.AccessPolicyRule.Permissions != nil {
		desiredPermissions = append(desiredPermissions, r.AccessPolicyRule.Permissions.Scopes...)
	}

	permissionIDs := permissions.
		Filter(desiredPermissions...).
		PermissionIDs()

	return msgraph.PreAuthorizedApplication{
		AppID:                  &clientId,
		DelegatedPermissionIDs: permissionIDs,
	}
}

type PrincipalType string

const (
	PrincipalTypeGroup            PrincipalType = "Group"
	PrincipalTypeServicePrincipal PrincipalType = "ServicePrincipal"
	PrincipalTypeUser             PrincipalType = "User"
)
