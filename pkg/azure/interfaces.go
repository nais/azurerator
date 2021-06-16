package azure

import (
	"context"
	"net/http"
	"time"

	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/azure/util/permissions"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/util/crypto"
)

type Client interface {
	Create(tx Transaction) (*ApplicationResult, error)
	Delete(tx Transaction) error
	Exists(tx Transaction) (*msgraph.Application, bool, error)
	Get(tx Transaction) (msgraph.Application, error)

	GetPreAuthorizedApps(tx Transaction) (*PreAuthorizedApps, error)
	GetServicePrincipal(tx Transaction) (msgraph.ServicePrincipal, error)

	AddCredentials(tx Transaction) (CredentialsSet, error)
	RotateCredentials(tx Transaction, existing CredentialsSet, inUse KeyIdsInUse) (CredentialsSet, error)
	PurgeCredentials(tx Transaction) error
	ValidateCredentials(tx Transaction, existing CredentialsSet) (bool, error)

	Update(tx Transaction) (*ApplicationResult, error)
}

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

	Delete(tx Transaction) error
	Exists(tx Transaction) (*msgraph.Application, bool, error)
	ExistsByFilter(ctx context.Context, filter Filter) (*msgraph.Application, bool, error)
	Get(tx Transaction) (msgraph.Application, error)
	GetByName(ctx context.Context, name DisplayName) (msgraph.Application, error)
	GetByClientId(ctx context.Context, id ClientId) (msgraph.Application, error)
	Patch(ctx context.Context, id ObjectId, application interface{}) error
	Register(tx Transaction) (*msgraph.Application, error)
	RemoveDisabledPermissions(tx Transaction, application msgraph.Application) error
	Update(tx Transaction) (*msgraph.Application, error)
}

type AppRoles interface {
	DescribeCreate(desired permissions.Permissions) []msgraph.AppRole
	DescribeUpdate(desired permissions.Permissions, existing []msgraph.AppRole) []msgraph.AppRole
}

type IdentifierUri interface {
	Set(tx Transaction) error
}

type OAuth2PermissionScope interface {
	DescribeCreate(desired permissions.Permissions) []msgraph.PermissionScope
	DescribeUpdate(desired permissions.Permissions, existing []msgraph.PermissionScope) []msgraph.PermissionScope
}

type ApplicationOwners interface {
	Process(tx Transaction, desired []msgraph.DirectoryObject) error
}

type RedirectUri interface {
	Update(tx Transaction) error
}

type AppRoleAssignmentsWithRoleId interface {
	ProcessForGroups(tx Transaction, assignees []Resource) error
	ProcessForServicePrincipals(tx Transaction, assignees []Resource) error
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
	Process(tx Transaction) error
}

type KeyCredential interface {
	Add(tx Transaction) (*AddedKeyCredentialSet, error)
	Rotate(tx Transaction, existing CredentialsSet, keyIdsInUse KeyIdsInUse) (*msgraph.KeyCredential, *crypto.Jwk, error)
	Purge(tx Transaction) error
	Validate(tx Transaction, existing CredentialsSet) (bool, error)
}

type OAuth2PermissionGrant interface {
	Process(tx Transaction) error
}

type PasswordCredential interface {
	Add(tx Transaction) (msgraph.PasswordCredential, error)
	Rotate(tx Transaction, existing CredentialsSet, keyIdsInUse KeyIdsInUse) (*msgraph.PasswordCredential, error)
	Purge(tx Transaction) error
	Validate(tx Transaction, existing CredentialsSet) (bool, error)
}

type PreAuthApps interface {
	Get(tx Transaction) (*PreAuthorizedApps, error)
	Process(tx Transaction, permissions permissions.Permissions) (*PreAuthorizedApps, error)
}

type ServicePrincipal interface {
	Owners() ServicePrincipalOwners
	Policies() ServicePrincipalPolicies

	Exists(ctx context.Context, id ClientId) (bool, msgraph.ServicePrincipal, error)
	Register(tx Transaction) (msgraph.ServicePrincipal, error)
	SetAppRoleAssignmentRequired(tx Transaction) error
	SetAppRoleAssignmentNotRequired(tx Transaction) error
}

type ServicePrincipalOwners interface {
	Process(tx Transaction, desired []msgraph.DirectoryObject) error
}

type ServicePrincipalPolicies interface {
	Process(tx Transaction) error
}

type Team interface {
	Owners() TeamOwners
	Groups() TeamGroups
}

type TeamOwners interface {
	Process(tx Transaction) error
}

type TeamGroups interface {
	Get(ctx context.Context) ([]msgraph.AppRoleAssignment, error)
}
