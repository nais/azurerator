package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/util/crypto"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Application tags
const (
	IntegratedAppTag string = "WindowsAzureActiveDirectoryIntegratedApp"
	IaCAppTag        string = "azurerator_appreg"
)

type application struct {
	client
}

type applicationResponse struct {
	Application   msgraph.Application
	KeyCredential msgraph.KeyCredential
	Jwk           crypto.Jwk
}

func (c client) application() application {
	return application{c}
}

func (a application) register(tx azure.Transaction) (applicationResponse, error) {
	key, jwk, err := a.keyCredential().new(tx.Instance)
	if err != nil {
		return applicationResponse{}, err
	}
	access := []msgraph.RequiredResourceAccess{
		a.requiredResourceAccess().microsoftGraph(),
	}
	appRoles := []msgraph.AppRole{
		a.appRoles().defaultRole(),
	}
	req := util.Application(a.defaultTemplate(tx.Instance)).
		Key(*key).
		ResourceAccess(access).
		GroupMembershipClaims(azure.GroupMembershipClaimApplicationGroup).
		AppRoles(appRoles).
		RedirectUris(util.GetReplyUrlsStringSlice(tx.Instance)).
		Build()
	app, err := a.graphClient.Applications().Request().Add(tx.Ctx, req)
	if err != nil {
		return applicationResponse{}, fmt.Errorf("registering application: %w", err)
	}

	return applicationResponse{
		Application:   *app,
		KeyCredential: *key,
		Jwk:           *jwk,
	}, nil
}

func (a application) delete(tx azure.Transaction) error {
	if err := a.graphClient.Applications().ID(tx.Instance.GetObjectId()).Request().Delete(tx.Ctx); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	return nil
}

func (a application) update(tx azure.Transaction) error {
	objectId := tx.Instance.GetObjectId()

	identifierUris := util.IdentifierUris(tx)

	defaultRole := a.appRoles().defaultRole()
	appRoles, err := a.appRoles().ensureExists(tx, defaultRole)
	if err != nil {
		return fmt.Errorf("updating approles for application: %w", err)
	}

	app := util.Application(a.defaultTemplate(tx.Instance)).
		IdentifierUriList(identifierUris).
		AppRoles(appRoles)

	if tx.Instance.Spec.Claims != nil && len(tx.Instance.Spec.Claims.Groups) > 0 {
		app.GroupMembershipClaims(azure.GroupMembershipClaimApplicationGroup)
	}

	app.Build()

	return a.patch(tx.Ctx, objectId, app)
}

func (a application) patch(ctx context.Context, id azure.ObjectId, application interface{}) error {
	req := a.graphClient.Applications().ID(id).Request()
	if err := req.JSONRequest(ctx, "PATCH", "", application, nil); err != nil {
		return fmt.Errorf("failed to update web application: %w", err)
	}
	return nil
}

func (a application) exists(tx azure.Transaction) (bool, error) {
	name := tx.Instance.GetUniqueName()
	return a.existsByFilter(tx.Ctx, util.FilterByName(name))
}

func (a application) existsByFilter(ctx context.Context, filter azure.Filter) (bool, error) {
	applications, err := a.getAll(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	return len(applications) > 0, nil
}

func (a application) getByName(ctx context.Context, name azure.DisplayName) (msgraph.Application, error) {
	application, err := a.getSingleByFilterOrError(ctx, util.FilterByName(name))
	if err != nil {
		return msgraph.Application{}, fmt.Errorf("fetching application with name '%s': %w", name, err)
	}
	return *application, nil
}

func (a application) getByClientId(ctx context.Context, id azure.ClientId) (msgraph.Application, error) {
	application, err := a.getSingleByFilterOrError(ctx, util.FilterByAppId(id))
	if err != nil {
		return msgraph.Application{}, fmt.Errorf("fetching application with clientId '%s': %w", id, err)
	}
	return *application, nil
}

func (a application) getAll(ctx context.Context, filters ...azure.Filter) ([]msgraph.Application, error) {
	r := a.graphClient.Applications().Request()
	r.Filter(util.MapFiltersToFilter(filters))
	applications, err := r.GetN(ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return nil, fmt.Errorf("failed to get list applications: %w", err)
	}
	return applications, nil
}

func (a application) defaultTemplate(resource v1.AzureAdApplication) *msgraph.Application {
	return &msgraph.Application{
		DisplayName:    ptr.String(resource.GetUniqueName()),
		SignInAudience: ptr.String("AzureADMyOrg"),
		Tags: []string{
			IaCAppTag,
			IntegratedAppTag,
		},
		API: &msgraph.APIApplication{
			AcceptMappedClaims:          ptr.Bool(true),
			RequestedAccessTokenVersion: ptr.Int(2),
			OAuth2PermissionScopes:      a.oAuth2PermissionScopes().defaultScopes(),
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
