package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/apis/v1alpha1"
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

type applicationResponse struct {
	Application   msgraph.Application
	KeyCredential msgraph.KeyCredential
	JwkPair       crypto.JwkPair
}

func (c client) registerApplication(tx azure.Transaction) (applicationResponse, error) {
	key, jwkPair, err := generateNewKeyCredentialFor(tx.Resource)
	if err != nil {
		return applicationResponse{}, err
	}
	preAuthApps, err := c.mapToMsGraphPreAuthApps(tx)
	if err != nil {
		return applicationResponse{}, err
	}
	api := toApiApplication(preAuthApps)
	applicationRequest := util.Application(defaultApplicationTemplate(tx.Resource)).Key(key).Api(api).Build()
	application, err := c.graphClient.Applications().Request().Add(tx.Ctx, applicationRequest)
	if err != nil {
		return applicationResponse{}, fmt.Errorf("failed to register application: %w", err)
	}
	return applicationResponse{
		Application:   *application,
		KeyCredential: key,
		JwkPair:       jwkPair,
	}, nil
}

func (c client) deleteApplication(tx azure.Transaction) error {
	if err := c.graphClient.Applications().ID(tx.Resource.Status.ObjectId).Request().Delete(tx.Ctx); err != nil {
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

func (c client) updateApplication(ctx context.Context, id string, application *msgraph.Application) error {
	if err := c.graphClient.Applications().ID(id).Request().Update(ctx, application); err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}
	return nil
}

func (c client) applicationExists(tx azure.Transaction) (bool, error) {
	name := tx.Resource.GetUniqueName()
	return c.applicationExistsByFilter(tx.Ctx, util.FilterByName(name))
}

func (c client) applicationExistsByFilter(ctx context.Context, filter azure.Filter) (bool, error) {
	applications, err := c.getAllApplications(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	return len(applications) > 0, nil
}

func (c client) getApplicationById(tx azure.Transaction) (msgraph.Application, error) {
	objectId := tx.Resource.Status.ObjectId
	application, err := c.graphClient.Applications().ID(objectId).Request().Get(tx.Ctx)
	if err != nil {
		return msgraph.Application{}, fmt.Errorf("failed to lookup azure application with ID '%s'", objectId)
	}
	return *application, nil
}

func (c client) getApplicationByName(ctx context.Context, name azure.DisplayName) (msgraph.Application, error) {
	applications, err := c.getAllApplications(ctx, util.FilterByName(name))
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

func (c client) getApplicationByClientId(ctx context.Context, id azure.ClientId) (msgraph.Application, error) {
	applications, err := c.getAllApplications(ctx, util.FilterByAppId(id))
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

func (c client) getAllApplications(ctx context.Context, filters ...azure.Filter) ([]msgraph.Application, error) {
	r := c.graphClient.Applications().Request()
	r.Filter(util.MapFiltersToFilter(filters))
	applications, err := r.GetN(ctx, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get list applications: %w", err)
	}
	return applications, nil
}

func defaultApplicationTemplate(resource v1alpha1.AzureAdApplication) *msgraph.Application {
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
			microsoftGraphResourceAccess(),
		},
		AppRoles: []msgraph.AppRole{
			defaultAppRole(),
		},
	}
}

func toApiApplication(preAuthApps []msgraph.PreAuthorizedApplication) *msgraph.APIApplication {
	return &msgraph.APIApplication{
		AcceptMappedClaims:          ptr.Bool(true),
		RequestedAccessTokenVersion: ptr.Int(2),
		Oauth2PermissionScopes:      toPermissionScopes(),
		PreAuthorizedApplications:   preAuthApps,
	}
}
