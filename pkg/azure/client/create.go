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
	IntegratedAppTag         string = "WindowsAzureActiveDirectoryIntegratedApp"
	IaCAppTag                string = "azurerator_appreg"
)

const (
	// OAuth2 permission scope that the web API application exposes to client applications
	OAuth2DefaultAccessScope string = "defaultaccess"
)

// Create registers a new AAD application
func (c client) Create(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	return c.registerApplication(ctx, credential)
}

// TODO - improve error handling
func (c client) registerApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	jwkPair, err := crypto.GenerateJwkPair(credential)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to generate JWK pair for application: %w", err)
	}
	keyCredential := util.ToKeyCredential(jwkPair)

	application, err := c.graphClient.Applications().Request().Add(ctx, toApplication(credential, keyCredential))
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to register application: %w", err)
	}

	// TODO - store serviceprincipal object ID
	servicePrincipal, err := c.graphBetaClient.ServicePrincipals().Request().Add(ctx, toServicePrincipal(application))
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to create service principal: %w", err)
	}

	// OAuth2 permission grants allows us to pre-approve this application for the defined scopes/permissions set.
	// This results in the enduser not having to manually consent whenever interacting with the application, e.g. during
	// an OIDC login flow.
	_, err = c.graphBetaClient.Oauth2PermissionGrants().Request().Add(ctx, toOAuth2PermissionGrants(servicePrincipal, c.config.PermissionGrantResourceId))
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to add oauth2 permission grants: %w", err)
	}

	passwordCredential, err := c.addPasswordCredential(ctx, *application.ID)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to update password credentials for application: %w", err)
	}

	if err := c.addApplicationIdentifierUri(ctx, *application); err != nil {
		return azure.Application{}, fmt.Errorf("failed to add application identifier URI: %w", err)
	}

	return azure.Application{
		Credentials: azure.Credentials{
			Public: azure.Public{
				ClientId: *application.AppID,
				Jwk:      jwkPair.Public,
			},
			Private: azure.Private{
				ClientId:     *application.AppID,
				ClientSecret: *passwordCredential.SecretText,
				Jwk:          jwkPair.Private,
			},
		},
		ClientId:         *application.AppID,
		ObjectId:         *application.ID,
		PasswordKeyId:    string(*passwordCredential.KeyID),
		CertificateKeyId: string(*keyCredential.KeyID),
	}, nil
}

func toApplication(credential v1alpha1.AzureAdCredential, keyCredential msgraph.KeyCredential) *msgraph.Application {
	oAuth2DefaultAccessScopeId := uuid.New().String()
	preAuthorizedApplications := mapToPreAuthorizedApplications(credential, oAuth2DefaultAccessScopeId)

	return &msgraph.Application{
		DisplayName:           ptr.String(credential.GetUniqueName()),
		GroupMembershipClaims: ptr.String("SecurityGroup"),
		KeyCredentials:        []msgraph.KeyCredential{keyCredential},
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
		API: &msgraph.APIApplication{
			AcceptMappedClaims:          ptr.Bool(true),
			RequestedAccessTokenVersion: ptr.Int(2),
			Oauth2PermissionScopes:      toOAuth2PermissionScopes(oAuth2DefaultAccessScopeId),
			PreAuthorizedApplications:   preAuthorizedApplications,
		},
	}
}

func mapToPreAuthorizedApplications(credential v1alpha1.AzureAdCredential, defaultAccessPermissionId string) []msgraph.PreAuthorizedApplication {
	var preAuthorizedApplications []msgraph.PreAuthorizedApplication
	for _, app := range credential.Spec.PreAuthorizedApplications {
		// TODO - lookup if ClientId is not present
		if len(app.ClientId) > 0 {
			preAuthorizedApplications = append(preAuthorizedApplications, msgraph.PreAuthorizedApplication{
				AppID: &app.ClientId,
				DelegatedPermissionIDs: []string{
					defaultAccessPermissionId,
				},
			})
		}
	}
	return preAuthorizedApplications
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

func toOAuth2PermissionScopes(id string) []msgraph.PermissionScope {
	defaultAccessScopeId := msgraph.UUID(id)
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

func toServicePrincipal(application *msgraph.Application) *msgraphbeta.ServicePrincipal {
	return &msgraphbeta.ServicePrincipal{
		AppID: application.AppID,
	}
}

func toOAuth2PermissionGrants(servicePrincipal *msgraphbeta.ServicePrincipal, permissionGrantResourceId string) *msgraphbeta.OAuth2PermissionGrant {
	// This field is required by Graph API, but isn't actually used...
	expiryTime := time.Date(0001, time.January, 1, 0, 0, 0, 0, time.UTC)
	return &msgraphbeta.OAuth2PermissionGrant{
		ClientID:    ptr.String(*servicePrincipal.ID),
		ConsentType: ptr.String("AllPrincipals"),
		ExpiryTime:  &expiryTime,
		ResourceID:  ptr.String(permissionGrantResourceId),
		Scope:       ptr.String("openid User.Read"),
	}
}
