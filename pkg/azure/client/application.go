package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
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

type applicationResponse struct {
	Application   msgraph.Application
	KeyCredential msgraph.KeyCredential
	JwkPair       crypto.JwkPair
}

func (c client) registerApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) (applicationResponse, error) {
	key, jwkPair, err := util.GenerateNewKeyCredentialFor(credential)
	if err != nil {
		return applicationResponse{}, err
	}
	api := c.toApiApplication(ctx, credential)
	applicationRequest := util.Application(defaultApplicationTemplate(credential)).Key(key).Api(api).Build()
	application, err := c.graphClient.Applications().Request().Add(ctx, applicationRequest)
	if err != nil {
		return applicationResponse{}, fmt.Errorf("failed to register application: %w", err)
	}
	return applicationResponse{
		Application:   *application,
		KeyCredential: key,
		JwkPair:       jwkPair,
	}, nil
}

func (c client) deleteApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) error {
	if err := c.graphClient.Applications().ID(credential.Status.ObjectId).Request().Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	return nil
}

func (c client) setApplicationIdentifierUri(ctx context.Context, application msgraph.Application) error {
	identifierUri := util.IdentifierUri(*application.AppID)
	app := util.EmptyApplication().IdentifierUri(identifierUri).Build()
	if err := c.updateApplication(ctx, *application.ID, app); err != nil {
		return fmt.Errorf("failed to add application identifier URI: %w", err)
	}
	return nil
}

func (c client) toApiApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) *msgraph.APIApplication {
	oAuth2DefaultAccessScopeId := uuid.New()
	preAuthorizedApplications := c.mapToPreAuthorizedApplications(ctx, credential, oAuth2DefaultAccessScopeId)
	return &msgraph.APIApplication{
		AcceptMappedClaims:          ptr.Bool(true),
		RequestedAccessTokenVersion: ptr.Int(2),
		Oauth2PermissionScopes:      toPermissionScopes(oAuth2DefaultAccessScopeId),
		PreAuthorizedApplications:   preAuthorizedApplications,
	}
}

func (c client) updateApplication(ctx context.Context, id string, application *msgraph.Application) error {
	if err := c.graphClient.Applications().ID(id).Request().Update(ctx, application); err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}
	return nil
}

func (c client) applicationExists(ctx context.Context, credential v1alpha1.AzureAdCredential) (bool, error) {
	applications, err := c.getAllApplications(ctx, util.FilterByName(credential.GetUniqueName()))
	if err != nil {
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	return len(applications) > 0, nil
}

func (c client) getApplicationById(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraph.Application, error) {
	objectId := credential.Status.ObjectId
	application, err := c.graphClient.Applications().ID(objectId).Request().Get(ctx)
	if err != nil {
		return msgraph.Application{}, fmt.Errorf("failed to lookup azure application with ID '%s'", objectId)
	}
	return *application, nil
}

func (c client) getApplicationByName(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraph.Application, error) {
	return c.getApplicationByStringName(ctx, credential.GetUniqueName())
}

func (c client) getApplicationByStringName(ctx context.Context, name string) (msgraph.Application, error) {
	applications, err := c.getAllApplications(ctx, util.FilterByName(name))
	if err != nil {
		return msgraph.Application{}, err
	}
	if len(applications) == 0 {
		return msgraph.Application{}, fmt.Errorf("could not find azure application with name '%s'", name)
	}
	if len(applications) > 1 {
		return msgraph.Application{}, fmt.Errorf("found more than one azure application with name '%s'", name)
	}
	return applications[0], nil
}

func (c client) getAllApplications(ctx context.Context, filters ...string) ([]msgraph.Application, error) {
	var applications []msgraph.Application

	r := c.graphClient.Applications().Request()
	r.Filter(util.MapFiltersToFilter(filters))
	applications, err := r.GetN(ctx, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get list applications: %w", err)
	}
	return applications, nil
}

func defaultApplicationTemplate(credential v1alpha1.AzureAdCredential) *msgraph.Application {
	return &msgraph.Application{
		DisplayName:           ptr.String(credential.GetUniqueName()),
		GroupMembershipClaims: ptr.String("SecurityGroup"),
		Web: &msgraph.WebApplication{
			LogoutURL:    ptr.String(credential.Spec.LogoutUrl),
			RedirectUris: util.GetReplyUrlsStringSlice(credential),
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
			microsoftGraphResourceAccess(),
		},
	}
}
