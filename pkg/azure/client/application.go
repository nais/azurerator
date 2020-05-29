package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/util/crypto"
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
	JwkPair       crypto.JwkPair
}

func (c client) application() application {
	return application{c}
}

func (a application) register(tx azure.Transaction) (applicationResponse, error) {
	key, jwkPair, err := a.keyCredential().new(tx.Instance)
	if err != nil {
		return applicationResponse{}, err
	}
	preAuthApps, err := a.preAuthApps().mapToMsGraph(tx)
	if err != nil {
		return applicationResponse{}, err
	}
	api := &msgraph.APIApplication{
		AcceptMappedClaims:          ptr.Bool(true),
		RequestedAccessTokenVersion: ptr.Int(2),
		Oauth2PermissionScopes:      a.oAuth2PermissionScopes().defaultScopes(),
		PreAuthorizedApplications:   preAuthApps,
	}
	req := util.Application(a.defaultTemplate(tx.Instance)).Key(*key).Api(api).Build()
	app, err := a.graphClient.Applications().Request().Add(tx.Ctx, req)
	if err != nil {
		return applicationResponse{}, fmt.Errorf("failed to register application: %w", err)
	}
	return applicationResponse{
		Application:   *app,
		KeyCredential: *key,
		JwkPair:       *jwkPair,
	}, nil
}

func (a application) delete(tx azure.Transaction) error {
	if err := a.graphClient.Applications().ID(tx.Instance.Status.ObjectId).Request().Delete(tx.Ctx); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	return nil
}

func (a application) update(ctx context.Context, id string, application *msgraph.Application) error {
	if err := a.graphClient.Applications().ID(id).Request().Update(ctx, application); err != nil {
		return fmt.Errorf("failed to update application: %w", err)
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

func (a application) getById(tx azure.Transaction) (msgraph.Application, error) {
	objectId := tx.Instance.Status.ObjectId
	application, err := a.graphClient.Applications().ID(objectId).Request().Get(tx.Ctx)
	if err != nil {
		return msgraph.Application{}, fmt.Errorf("failed to lookup azure application with ID '%s'", objectId)
	}
	return *application, nil
}

func (a application) getByName(ctx context.Context, name azure.DisplayName) (msgraph.Application, error) {
	applications, err := a.getAll(ctx, util.FilterByName(name))
	if err != nil {
		return msgraph.Application{}, err
	}
	switch {
	case len(applications) == 0:
		return msgraph.Application{}, fmt.Errorf("could not find azure application with name '%s'", name)
	case len(applications) > 1:
		return msgraph.Application{}, fmt.Errorf("found more than one azure application with name '%s'", name)
	default:
		return applications[0], nil
	}
}

func (a application) getByClientId(ctx context.Context, id azure.ClientId) (msgraph.Application, error) {
	applications, err := a.getAll(ctx, util.FilterByAppId(id))
	if err != nil {
		return msgraph.Application{}, err
	}
	switch {
	case len(applications) == 0:
		return msgraph.Application{}, fmt.Errorf("could not find azure application with clientId '%s'", id)
	case len(applications) > 1:
		return msgraph.Application{}, fmt.Errorf("found more than one azure application with clientId '%s'", id)
	default:
		return applications[0], nil
	}
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

func (a application) defaultTemplate(resource v1alpha1.AzureAdApplication) *msgraph.Application {
	return &msgraph.Application{
		DisplayName:           ptr.String(resource.GetUniqueName()),
		GroupMembershipClaims: ptr.String("SecurityGroup"),
		Web: &msgraph.WebApplication{
			LogoutURL:    ptr.String(resource.Spec.LogoutUrl),
			RedirectUris: util.GetReplyUrlsStringSlice(resource),
			ImplicitGrantSettings: &msgraph.ImplicitGrantSettings{
				EnableIDTokenIssuance:     ptr.Bool(false),
				EnableAccessTokenIssuance: ptr.Bool(false),
			},
		},
		SignInAudience: ptr.String("AzureADMyOrg"),
		Tags: []string{
			IaCAppTag,
			IntegratedAppTag,
		},
		RequiredResourceAccess: []msgraph.RequiredResourceAccess{
			a.requiredResourceAccess().microsoftGraph(),
		},
		AppRoles: []msgraph.AppRole{
			a.appRoles().defaultRole(),
		},
	}
}
