package application

import (
	"context"
	"fmt"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/azure/util/permissions"
)

// Application tags
const (
	IntegratedAppTag string = "WindowsAzureActiveDirectoryIntegratedApp"
	IaCAppTag        string = "azurerator_appreg"
)

type Application struct {
	azure.RuntimeClient
}

func NewApplication(runtimeClient azure.RuntimeClient) azure.Application {
	return Application{RuntimeClient: runtimeClient}
}

func (a Application) AppRoles() azure.AppRoles {
	return newAppRoles(a)
}

func (a Application) IdentifierUri() azure.IdentifierUri {
	return newIdentifierUri(a)
}

func (a Application) OAuth2PermissionScopes() azure.OAuth2PermissionScope {
	return newOAuth2PermissionScopes(a)
}

func (a Application) Owners() azure.ApplicationOwners {
	return newOwners(a.RuntimeClient)
}

func (a Application) RedirectUri() azure.RedirectUri {
	return newRedirectUri(a)
}

func (a Application) requiredResourceAccess() requiredResourceAccess {
	return newRequiredResourceAccess(a)
}

func (a Application) Exists(tx azure.Transaction) (*msgraph.Application, bool, error) {
	name := kubernetes.UniformResourceName(&tx.Instance)
	return a.ExistsByFilter(tx.Ctx, util.FilterByName(name))
}

func (a Application) Delete(tx azure.Transaction) error {
	if err := a.GraphClient().Applications().ID(tx.Instance.GetObjectId()).Request().Delete(tx.Ctx); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	return nil
}

func (a Application) Register(tx azure.Transaction) (*msgraph.Application, error) {
	access := []msgraph.RequiredResourceAccess{
		a.requiredResourceAccess().microsoftGraph(),
	}
	desiredPermissions := permissions.GenerateDesiredPermissionSet(tx.Instance)
	roles := a.AppRoles().DescribeCreate(desiredPermissions)
	scopes := a.OAuth2PermissionScopes().DescribeCreate(desiredPermissions)

	req := util.Application(a.defaultTemplate(tx.Instance)).
		ResourceAccess(access).
		GroupMembershipClaims(azure.GroupMembershipClaimApplicationGroup).
		AppRoles(roles).
		RedirectUris(util.GetReplyUrlsStringSlice(tx.Instance)).
		PermissionScopes(scopes).
		Build()

	app, err := a.GraphClient().Applications().Request().Add(tx.Ctx, req)
	if err != nil {
		return nil, fmt.Errorf("registering application: %w", err)
	}

	return app, nil
}

func (a Application) Update(tx azure.Transaction) (*msgraph.Application, error) {
	objectId := tx.Instance.GetObjectId()
	clientId := tx.Instance.GetClientId()
	identifierUris := util.IdentifierUris(tx)

	actualApp, err := a.GetByClientId(tx.Ctx, clientId)
	if err != nil {
		return nil, err
	}

	desiredPermissions := permissions.GenerateDesiredPermissionSetPreserveExisting(tx.Instance, actualApp)

	existingRoles := actualApp.AppRoles
	roles := a.AppRoles().DescribeUpdate(desiredPermissions, existingRoles)

	existingScopes := actualApp.API.OAuth2PermissionScopes
	scopes := a.OAuth2PermissionScopes().DescribeUpdate(desiredPermissions, existingScopes)

	builder := util.Application(a.defaultTemplate(tx.Instance)).
		IdentifierUriList(identifierUris).
		PermissionScopes(scopes).
		AppRoles(roles)

	groupClaimsIsDefined := tx.Instance.Spec.Claims != nil && len(tx.Instance.Spec.Claims.Groups) > 0

	// todo: remove 'groupClaimsIsDefined' predicate after grace period
	if a.Config().Features.GroupsAssignment.Enabled && groupClaimsIsDefined {
		builder.GroupMembershipClaims(azure.GroupMembershipClaimApplicationGroup)
	}

	app := builder.Build()
	return app, a.Patch(tx.Ctx, objectId, app)
}

func (a Application) Patch(ctx context.Context, id azure.ObjectId, application interface{}) error {
	req := a.GraphClient().Applications().ID(id).Request()
	if err := req.JSONRequest(ctx, "PATCH", "", application, nil); err != nil {
		return fmt.Errorf("failed to update web application: %w", err)
	}
	return nil
}

func (a Application) ExistsByFilter(ctx context.Context, filter azure.Filter) (*msgraph.Application, bool, error) {
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

func (a Application) Get(tx azure.Transaction) (msgraph.Application, error) {
	return a.GetByName(tx.Ctx, kubernetes.UniformResourceName(&tx.Instance))
}

func (a Application) GetByName(ctx context.Context, name azure.DisplayName) (msgraph.Application, error) {
	application, err := a.getSingleByFilterOrError(ctx, util.FilterByName(name))
	if err != nil {
		return msgraph.Application{}, fmt.Errorf("fetching application with name '%s': %w", name, err)
	}
	return *application, nil
}

func (a Application) GetByClientId(ctx context.Context, id azure.ClientId) (msgraph.Application, error) {
	application, err := a.getSingleByFilterOrError(ctx, util.FilterByAppId(id))
	if err != nil {
		return msgraph.Application{}, fmt.Errorf("fetching application with clientId '%s': %w", id, err)
	}
	return *application, nil
}

func (a Application) getAll(ctx context.Context, filters ...azure.Filter) ([]msgraph.Application, error) {
	r := a.GraphClient().Applications().Request()
	r.Filter(util.MapFiltersToFilter(filters))
	applications, err := r.GetN(ctx, a.RuntimeClient.MaxNumberOfPagesToFetch())
	if err != nil {
		return nil, fmt.Errorf("failed to get list applications: %w", err)
	}
	return applications, nil
}

func (a Application) defaultTemplate(resource v1.AzureAdApplication) *msgraph.Application {
	return &msgraph.Application{
		DisplayName:    ptr.String(kubernetes.UniformResourceName(&resource)),
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
			LogoutURL: ptr.String(resource.Spec.LogoutUrl),
			ImplicitGrantSettings: &msgraph.ImplicitGrantSettings{
				EnableIDTokenIssuance:     ptr.Bool(false),
				EnableAccessTokenIssuance: ptr.Bool(false),
			},
		},
	}
}

func (a Application) getSingleByFilterOrError(ctx context.Context, filter azure.Filter) (*msgraph.Application, error) {
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
