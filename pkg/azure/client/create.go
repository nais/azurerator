package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/util/crypto"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Application tags
const (
	IntegratedAppTag string = "WindowsAzureActiveDirectoryIntegratedApp"
	IaCAppTag        string = "azurerator_appreg"
)

const (
	// OAuth2 permission scope that the web API application exposes to client applications
	OAuth2DefaultAccessScope string = "defaultaccess"
)

type applicationResponse struct {
	Application   msgraph.Application
	KeyCredential msgraph.KeyCredential
	JwkPair       crypto.JwkPair
}

// Create registers a new AAD application with all the required accompanying resources
// TODO - improve error handling
func (c client) Create(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	applicationResponse, err := c.registerApplication(ctx, credential)
	if err != nil {
		return azure.Application{}, err
	}
	passwordCredential, err := c.addPasswordCredential(ctx, *applicationResponse.Application.ID)
	if err != nil {
		return azure.Application{}, err
	}
	servicePrincipal, err := c.registerServicePrincipal(ctx, applicationResponse.Application)
	if err != nil {
		return azure.Application{}, err
	}
	if err := c.setApplicationIdentifierUri(ctx, applicationResponse.Application); err != nil {
		return azure.Application{}, err
	}
	if err := c.registerOAuth2PermissionGrants(ctx, servicePrincipal); err != nil {
		return azure.Application{}, err
	}
	return azure.Application{
		Credentials: azure.Credentials{
			Public: azure.Public{
				ClientId: *applicationResponse.Application.AppID,
				Jwk:      applicationResponse.JwkPair.Public,
			},
			Private: azure.Private{
				ClientId:     *applicationResponse.Application.AppID,
				ClientSecret: *passwordCredential.SecretText,
				Jwk:          applicationResponse.JwkPair.Private,
			},
		},
		ClientId:           *applicationResponse.Application.AppID,
		ObjectId:           *applicationResponse.Application.ID,
		PasswordKeyId:      string(*passwordCredential.KeyID),
		CertificateKeyId:   string(*applicationResponse.KeyCredential.KeyID),
		ServicePrincipalId: *servicePrincipal.ID,
	}, nil
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

// TODO - should attempt to register on update as well
func (c client) registerServicePrincipal(ctx context.Context, application msgraph.Application) (msgraphbeta.ServicePrincipal, error) {
	servicePrincipal, err := c.graphBetaClient.ServicePrincipals().Request().Add(ctx, toServicePrincipal(application))
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to register service principal: %w", err)
	}
	return *servicePrincipal, nil
}

// TODO - should attempt to register on update as well
func (c client) registerOAuth2PermissionGrants(ctx context.Context, principal msgraphbeta.ServicePrincipal) error {
	_, err := c.graphBetaClient.Oauth2PermissionGrants().Request().Add(ctx, toOAuth2PermissionGrants(&principal, c.config.PermissionGrantResourceId))
	if err != nil {
		return fmt.Errorf("failed to register oauth2 permission grants: %w", err)
	}
	return nil
}

func (c client) toApiApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) *msgraph.APIApplication {
	oAuth2DefaultAccessScopeId := uuid.New()
	preAuthorizedApplications := c.mapToPreAuthorizedApplications(ctx, credential, oAuth2DefaultAccessScopeId)
	return toApiApplication(oAuth2DefaultAccessScopeId, preAuthorizedApplications)
}

func (c client) mapToPreAuthorizedApplications(ctx context.Context, credential v1alpha1.AzureAdCredential, defaultAccessPermissionId uuid.UUID) []msgraph.PreAuthorizedApplication {
	var preAuthorizedApplications []msgraph.PreAuthorizedApplication
	for _, app := range credential.Spec.PreAuthorizedApplications {
		clientId, err := c.getClientId(ctx, app)
		if err != nil {
			// TODO - currently best effort. should separate between technical and functional (e.g. app doesnt exist in AAD) errors
			fmt.Printf("%v\n", err)
			continue
		}
		preAuthorizedApplication := msgraph.PreAuthorizedApplication{
			AppID: &clientId,
			DelegatedPermissionIDs: []string{
				defaultAccessPermissionId.String(),
			},
		}
		preAuthorizedApplications = append(preAuthorizedApplications, preAuthorizedApplication)
	}
	return preAuthorizedApplications
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
			microsoftGraphApiPermissions(),
		},
	}
}

func toApiApplication(permissionScopeId uuid.UUID, preAuthorizedApplications []msgraph.PreAuthorizedApplication) *msgraph.APIApplication {
	return &msgraph.APIApplication{
		AcceptMappedClaims:          ptr.Bool(true),
		RequestedAccessTokenVersion: ptr.Int(2),
		Oauth2PermissionScopes:      toPermissionScopes(permissionScopeId),
		PreAuthorizedApplications:   preAuthorizedApplications,
	}
}

func microsoftGraphApiPermissions() msgraph.RequiredResourceAccess {
	userReadScopeId := msgraph.UUID("e1fe6dd8-ba31-4d61-89e7-88639da4683d") // User.Read
	openidScopeId := msgraph.UUID("37f7f235-527c-4136-accd-4a02d197296e")   // openid
	return msgraph.RequiredResourceAccess{
		ResourceAppID: ptr.String("00000003-0000-0000-c000-000000000000"),
		ResourceAccess: []msgraph.ResourceAccess{
			{
				ID:   &userReadScopeId,
				Type: ptr.String("Scope"),
			},
			{
				ID:   &openidScopeId,
				Type: ptr.String("Scope"),
			},
		},
	}
}

func toPermissionScopes(id uuid.UUID) []msgraph.PermissionScope {
	defaultAccessScopeId := msgraph.UUID(id.String())
	return []msgraph.PermissionScope{
		{
			AdminConsentDescription: ptr.String(fmt.Sprintf("Gives adminconsent for scope %s", OAuth2DefaultAccessScope)),
			AdminConsentDisplayName: ptr.String(fmt.Sprintf("Adminconsent for scope %s", OAuth2DefaultAccessScope)),
			ID:                      &defaultAccessScopeId,
			IsEnabled:               ptr.Bool(true),
			Type:                    ptr.String("User"),
			Value:                   ptr.String(OAuth2DefaultAccessScope),
		},
	}
}

func toServicePrincipal(application msgraph.Application) *msgraphbeta.ServicePrincipal {
	return &msgraphbeta.ServicePrincipal{
		AppID: application.AppID,
	}
}

// OAuth2 permission grants allows us to pre-approve this application for the defined scopes/permissions set.
// This results in the enduser not having to manually consent whenever interacting with the application, e.g. during
// an OIDC login flow.
func toOAuth2PermissionGrants(servicePrincipal *msgraphbeta.ServicePrincipal, permissionGrantResourceId string) *msgraphbeta.OAuth2PermissionGrant {
	// This field is required by Graph API, but isn't actually used as per 2020-04-29.
	// https://docs.microsoft.com/en-us/graph/api/resources/oauth2permissiongrant?view=graph-rest-beta
	expiryTime := time.Date(0001, time.January, 1, 0, 0, 0, 0, time.UTC)
	return &msgraphbeta.OAuth2PermissionGrant{
		ClientID:    ptr.String(*servicePrincipal.ID),
		ConsentType: ptr.String("AllPrincipals"),
		ExpiryTime:  &expiryTime,
		ResourceID:  ptr.String(permissionGrantResourceId),
		Scope:       ptr.String("openid User.Read"),
	}
}
