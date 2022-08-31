package application

import (
	"context"
	"fmt"

	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application/approle"
	"github.com/nais/azureator/pkg/azure/client/application/identifieruri"
	"github.com/nais/azureator/pkg/azure/client/application/optionalclaims"
	"github.com/nais/azureator/pkg/azure/client/application/owners"
	"github.com/nais/azureator/pkg/azure/client/application/permissionscope"
	"github.com/nais/azureator/pkg/azure/client/application/redirecturi"
	"github.com/nais/azureator/pkg/azure/client/application/requiredresourceaccess"
	"github.com/nais/azureator/pkg/azure/permissions"
	"github.com/nais/azureator/pkg/azure/transaction"
	"github.com/nais/azureator/pkg/azure/util"
)

// Application tags
const (
	IntegratedAppTag string = "WindowsAzureActiveDirectoryIntegratedApp"
	IaCAppTag        string = "azurerator_appreg"
)

type Application interface {
	AppRoles() approle.AppRoles
	IdentifierUri() identifieruri.IdentifierUri
	OAuth2PermissionScopes() permissionscope.OAuth2PermissionScope
	Owners() owners.Owners
	RedirectUri() redirecturi.RedirectUri

	Delete(tx transaction.Transaction) error
	Exists(tx transaction.Transaction) (*msgraph.Application, bool, error)
	ExistsByFilter(ctx context.Context, filter azure.Filter) (*msgraph.Application, bool, error)
	Get(tx transaction.Transaction) (msgraph.Application, error)
	GetByName(ctx context.Context, name azure.DisplayName) (msgraph.Application, error)
	GetByClientId(ctx context.Context, id azure.ClientId) (msgraph.Application, error)
	Patch(ctx context.Context, id azure.ObjectId, application any) error
	Register(tx transaction.Transaction) (*msgraph.Application, error)
	RemoveDisabledPermissions(tx transaction.Transaction, application msgraph.Application) error
	Update(tx transaction.Transaction) (*msgraph.Application, error)
}

type application struct {
	azure.RuntimeClient
}

func NewApplication(runtimeClient azure.RuntimeClient) Application {
	return application{RuntimeClient: runtimeClient}
}

func (a application) AppRoles() approle.AppRoles {
	return approle.NewAppRoles()
}

func (a application) IdentifierUri() identifieruri.IdentifierUri {
	return identifieruri.NewIdentifierUri(a)
}

func (a application) OAuth2PermissionScopes() permissionscope.OAuth2PermissionScope {
	return permissionscope.NewOAuth2PermissionScopes()
}

func (a application) OptionalClaims() optionalclaims.OptionalClaims {
	return optionalclaims.NewOptionalClaims()
}

func (a application) Owners() owners.Owners {
	return owners.NewOwners(a.RuntimeClient)
}

func (a application) RedirectUri() redirecturi.RedirectUri {
	return redirecturi.NewRedirectUri(a)
}

func (a application) RequiredResourceAccess() requiredresourceaccess.RequiredResourceAccess {
	return requiredresourceaccess.NewRequiredResourceAccess()
}

func (a application) Exists(tx transaction.Transaction) (*msgraph.Application, bool, error) {
	name := kubernetes.UniformResourceName(&tx.Instance, tx.ClusterName)
	return a.ExistsByFilter(tx.Ctx, util.FilterByName(name))
}

func (a application) Delete(tx transaction.Transaction) error {
	if err := a.GraphClient().Applications().ID(tx.Instance.GetObjectId()).Request().Delete(tx.Ctx); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	return nil
}

func (a application) Register(tx transaction.Transaction) (*msgraph.Application, error) {
	access := []msgraph.RequiredResourceAccess{
		a.RequiredResourceAccess().MicrosoftGraph(),
	}
	desiredPermissions := permissions.GenerateDesiredPermissionSet(tx.Instance)

	roles := a.AppRoles().DescribeCreate(desiredPermissions)
	roles.Log(tx.Log)

	scopes := a.OAuth2PermissionScopes().DescribeCreate(desiredPermissions)
	scopes.Log(tx.Log)

	redirectUris := redirecturi.ReplyUrlsToStringSlice(tx.Instance)

	optionalClaims := a.OptionalClaims().DescribeCreate()

	req := util.Application(a.defaultTemplate(tx)).
		AppRoles(roles.GetResult()).
		GroupMembershipClaims(a.Config().Features.GroupMembershipClaim.Default).
		OptionalClaims(optionalClaims).
		PermissionScopes(scopes.GetResult()).
		RedirectUris(redirectUris, tx.Instance).
		ResourceAccess(access).
		Build()

	app, err := a.GraphClient().Applications().Request().Add(tx.Ctx, req)
	if err != nil {
		return nil, fmt.Errorf("registering application: %w", err)
	}

	return app, nil
}

func (a application) Update(tx transaction.Transaction) (*msgraph.Application, error) {
	objectId := tx.Instance.GetObjectId()
	clientId := tx.Instance.GetClientId()

	actualApp, err := a.GetByClientId(tx.Ctx, clientId)
	if err != nil {
		return nil, err
	}

	desiredPermissions := permissions.GenerateDesiredPermissionSetPreserveExisting(tx.Instance, actualApp)

	existingRoles := actualApp.AppRoles
	roles := a.AppRoles().DescribeUpdate(desiredPermissions, existingRoles)
	roles.Log(tx.Log)

	existingScopes := actualApp.API.OAuth2PermissionScopes
	scopes := a.OAuth2PermissionScopes().DescribeUpdate(desiredPermissions, existingScopes)
	scopes.Log(tx.Log)

	identifierUris := identifieruri.DescribeUpdate(tx.Instance, actualApp.IdentifierUris, tx.ClusterName)
	optionalClaims := a.OptionalClaims().DescribeUpdate(actualApp)
	app := util.Application(a.defaultTemplate(tx)).
		AppRoles(roles.GetResult()).
		IdentifierUriList(identifierUris).
		OptionalClaims(optionalClaims).
		PermissionScopes(scopes.GetResult()).
		Build()

	return app, a.Patch(tx.Ctx, objectId, app)
}

func (a application) Patch(ctx context.Context, id azure.ObjectId, application any) error {
	req := a.GraphClient().Applications().ID(id).Request()
	if err := req.JSONRequest(ctx, "PATCH", "", application, nil); err != nil {
		return fmt.Errorf("failed to update web application: %w", err)
	}
	return nil
}

func (a application) ExistsByFilter(ctx context.Context, filter azure.Filter) (*msgraph.Application, bool, error) {
	applications, err := a.getAll(ctx, filter)
	if err != nil {
		return nil, false, err
	}
	switch {
	case len(applications) == 0:
		return nil, false, nil
	case len(applications) > 1:
		return nil, true, fmt.Errorf("found more than one matching azure application")
	default:
		return &applications[0], true, nil
	}
}

func (a application) Get(tx transaction.Transaction) (msgraph.Application, error) {
	return a.GetByName(tx.Ctx, kubernetes.UniformResourceName(&tx.Instance, tx.ClusterName))
}

func (a application) GetByName(ctx context.Context, name azure.DisplayName) (msgraph.Application, error) {
	application, err := a.getSingleByFilterOrError(ctx, util.FilterByName(name))
	if err != nil {
		return msgraph.Application{}, fmt.Errorf("fetching application with name '%s': %w", name, err)
	}
	return *application, nil
}

func (a application) GetByClientId(ctx context.Context, id azure.ClientId) (msgraph.Application, error) {
	application, err := a.getSingleByFilterOrError(ctx, util.FilterByAppId(id))
	if err != nil {
		return msgraph.Application{}, fmt.Errorf("fetching application with clientId '%s': %w", id, err)
	}
	return *application, nil
}

// - we _CANNOT_ delete a disabled PermissionScope that has been granted to any pre-authorized app
// - we _CAN_ however delete a disabled AppRole _without_ removing the associated approleassignments first
func (a application) RemoveDisabledPermissions(tx transaction.Transaction, application msgraph.Application) error {
	objectId := tx.Instance.GetObjectId()

	scopes := permissionscope.RemoveDisabled(application)
	roles := approle.RemoveDisabled(application)

	patchedApp := util.EmptyApplication().
		PermissionScopes(scopes).
		AppRoles(roles)

	if err := a.Patch(tx.Ctx, objectId, patchedApp); err != nil {
		return fmt.Errorf("removing disabled permissions: %w", err)
	}

	return nil
}

func (a application) getAll(ctx context.Context, filters ...azure.Filter) ([]msgraph.Application, error) {
	r := a.GraphClient().Applications().Request()
	r.Filter(util.MapFiltersToFilter(filters))
	applications, err := r.GetN(ctx, a.RuntimeClient.MaxNumberOfPagesToFetch())
	if err != nil {
		return nil, fmt.Errorf("failed to get list applications: %w", err)
	}
	return applications, nil
}

func (a application) defaultTemplate(tx transaction.Transaction) *msgraph.Application {
	return &msgraph.Application{
		DisplayName:    ptr.String(kubernetes.UniformResourceName(&tx.Instance, tx.ClusterName)),
		SignInAudience: ptr.String("AzureADMyOrg"),
		Tags: []string{
			IaCAppTag,
			IntegratedAppTag,
		},
		API: &msgraph.APIApplication{
			AcceptMappedClaims:          ptr.Bool(true),
			RequestedAccessTokenVersion: ptr.Int(2),
		},
		Web: &msgraph.WebApplication{
			LogoutURL: ptr.String(tx.Instance.Spec.LogoutUrl),
			ImplicitGrantSettings: &msgraph.ImplicitGrantSettings{
				EnableIDTokenIssuance:     ptr.Bool(false),
				EnableAccessTokenIssuance: ptr.Bool(false),
			},
		},
	}
}

func (a application) getSingleByFilterOrError(ctx context.Context, filter azure.Filter) (*msgraph.Application, error) {
	applications, err := a.getAll(ctx, filter)
	if err != nil {
		return nil, err
	}
	switch {
	case len(applications) == 0:
		return nil, fmt.Errorf("no matching azure applications found")
	case len(applications) > 1:
		return nil, fmt.Errorf("found more than one matching azure application")
	default:
		return &applications[0], nil
	}
}
