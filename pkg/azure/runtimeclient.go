package azure

import (
	"context"
	"net/http"
	"time"

	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/azure/client/application/approle"
	"github.com/nais/azureator/pkg/azure/client/application/permissionscope"
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/resource"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/azure/transaction"
	"github.com/nais/azureator/pkg/azure/util/permissions"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/util/crypto"
)

type RuntimeClient interface {
	Config() *config.AzureConfig
	GraphClient() *msgraph.GraphServiceRequestBuilder
	HttpClient() *http.Client

	DelayIntervalBetweenModifications() time.Duration
	DelayIntervalBetweenCreations() time.Duration
	MaxNumberOfPagesToFetch() int

	Application() Application
	AppRoleAssignments(roleId msgraph.UUID, targetId ObjectId) AppRoleAssignmentsWithRoleId
	AppRoleAssignmentsNoRoleId(targetId ObjectId) AppRoleAssignments
	Groups() Groups
	KeyCredential() KeyCredential
	OAuth2PermissionGrant() OAuth2PermissionGrant
	PasswordCredential() PasswordCredential
	PreAuthApps() PreAuthApps
	ServicePrincipal() ServicePrincipal
	Team() Team
}

type Application interface {
	AppRoles() AppRoles
	IdentifierUri() IdentifierUri
	OAuth2PermissionScopes() OAuth2PermissionScope
	Owners() ApplicationOwners
	RedirectUri() RedirectUri

	Delete(tx transaction.Transaction) error
	Exists(tx transaction.Transaction) (*msgraph.Application, bool, error)
	ExistsByFilter(ctx context.Context, filter Filter) (*msgraph.Application, bool, error)
	Get(tx transaction.Transaction) (msgraph.Application, error)
	GetByName(ctx context.Context, name DisplayName) (msgraph.Application, error)
	GetByClientId(ctx context.Context, id ClientId) (msgraph.Application, error)
	Patch(ctx context.Context, id ObjectId, application interface{}) error
	Register(tx transaction.Transaction) (*msgraph.Application, error)
	RemoveDisabledPermissions(tx transaction.Transaction, application msgraph.Application) error
	Update(tx transaction.Transaction) (*msgraph.Application, error)
}

type AppRoles interface {
	DescribeCreate(desired permissions.Permissions) approle.CreateResult
	DescribeUpdate(desired permissions.Permissions, existing []msgraph.AppRole) approle.UpdateResult
}

type IdentifierUri interface {
	Set(tx transaction.Transaction) error
}

type OAuth2PermissionScope interface {
	DescribeCreate(desired permissions.Permissions) permissionscope.CreateResult
	DescribeUpdate(desired permissions.Permissions, existing []msgraph.PermissionScope) permissionscope.UpdateResult
}

type ApplicationOwners interface {
	Process(tx transaction.Transaction, desired []msgraph.DirectoryObject) error
}

type RedirectUri interface {
	Update(tx transaction.Transaction) error
}

type AppRoleAssignmentsWithRoleId interface {
	ProcessForGroups(tx transaction.Transaction, assignees []resource.Resource) error
	ProcessForServicePrincipals(tx transaction.Transaction, assignees []resource.Resource) error
}

type AppRoleAssignments interface {
	GetAll(ctx context.Context) ([]msgraph.AppRoleAssignment, error)
	GetAllGroups(ctx context.Context) ([]msgraph.AppRoleAssignment, error)
	GetAllServicePrincipals(ctx context.Context) ([]msgraph.AppRoleAssignment, error)
	LogFields() log.Fields
	Request() *msgraph.ServicePrincipalAppRoleAssignedToCollectionRequest
	TargetId() ObjectId
}

type Groups interface {
	GetOwnersFor(ctx context.Context, groupId string) ([]msgraph.DirectoryObject, error)
	Process(tx transaction.Transaction) error
}

type KeyCredential interface {
	Add(tx transaction.Transaction) (*credentials.AddedKeyCredentialSet, error)
	Rotate(tx transaction.Transaction, existing credentials.Set, keyIdsInUse credentials.KeyIdsInUse) (*msgraph.KeyCredential, *crypto.Jwk, error)
	Purge(tx transaction.Transaction) error
	Validate(tx transaction.Transaction, existing credentials.Set) (bool, error)
}

type OAuth2PermissionGrant interface {
	Process(tx transaction.Transaction) error
}

type PasswordCredential interface {
	Add(tx transaction.Transaction) (msgraph.PasswordCredential, error)
	Rotate(tx transaction.Transaction, existing credentials.Set, keyIdsInUse credentials.KeyIdsInUse) (*msgraph.PasswordCredential, error)
	Purge(tx transaction.Transaction) error
	Validate(tx transaction.Transaction, existing credentials.Set) (bool, error)
}

type PreAuthApps interface {
	Get(tx transaction.Transaction) (*result.PreAuthorizedApps, error)
	Process(tx transaction.Transaction, permissions permissions.Permissions) (*result.PreAuthorizedApps, error)
}

type ServicePrincipal interface {
	Owners() ServicePrincipalOwners
	Policies() ServicePrincipalPolicies

	Exists(ctx context.Context, id ClientId) (bool, msgraph.ServicePrincipal, error)
	Register(tx transaction.Transaction) (msgraph.ServicePrincipal, error)
	SetAppRoleAssignmentRequired(tx transaction.Transaction) error
	SetAppRoleAssignmentNotRequired(tx transaction.Transaction) error
}

type ServicePrincipalOwners interface {
	Process(tx transaction.Transaction, desired []msgraph.DirectoryObject) error
}

type ServicePrincipalPolicies interface {
	Process(tx transaction.Transaction) error
}

type Team interface {
	Owners() TeamOwners
	Groups() TeamGroups
}

type TeamOwners interface {
	Process(tx transaction.Transaction) error
}

type TeamGroups interface {
	Get(ctx context.Context) ([]msgraph.AppRoleAssignment, error)
}
