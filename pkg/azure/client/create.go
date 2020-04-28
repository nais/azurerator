package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

const (
	OAuth2DefaultAccessScope string = "defaultaccess"
)

// Create registers a new AAD application
func (c client) Create(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	return c.registerApplication(ctx, credential)
}

// TODO - add grants, owners, preauthorizedapps/approles
func (c client) registerApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	jwkPair, err := crypto.GenerateJwkPair(credential)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to generate JWK pair for application: %w", err)
	}
	keyCredential := util.CreateKeyCredential(jwkPair)

	application, err := c.graphClient.Applications().Request().Add(ctx, createApplication(credential, keyCredential))
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to register application: %w", err)
	}

	passwordCredential, err := c.addPasswordCredential(ctx, *application.ID)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to update password credentials for application: %w", err)
	}

	// TODO - add identifierUri to application

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

func createApplication(credential v1alpha1.AzureAdCredential, keyCredential msgraph.KeyCredential) *msgraph.Application {
	oauthDefaultAccessScopeId := uuid.New().String()
	msgraphOauthDefaultAccessScopeId := msgraph.UUID(oauthDefaultAccessScopeId)
	preAuthorizedApplications := mapToPreAuthorizedApplications(credential, oauthDefaultAccessScopeId)

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
		SignInAudience: ptr.String(SignInAudience),
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
			Oauth2PermissionScopes: []msgraph.PermissionScope{
				{
					AdminConsentDescription: ptr.String(fmt.Sprintf("Gives adminconsent for scope %s", OAuth2DefaultAccessScope)),
					AdminConsentDisplayName: ptr.String(fmt.Sprintf("Adminconsent for scope %s", OAuth2DefaultAccessScope)),
					ID:                      &msgraphOauthDefaultAccessScopeId,
					IsEnabled:               ptr.Bool(true),
					Type:                    ptr.String("User"),
					Value:                   ptr.String(OAuth2DefaultAccessScope),
				},
			},
			PreAuthorizedApplications: preAuthorizedApplications,
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
	return msgraph.RequiredResourceAccess{
		ResourceAppID: ptr.String("00000003-0000-0000-c000-000000000000"),
		ResourceAccess: []msgraph.ResourceAccess{
			{
				ID:   &userReadScopeId,
				Type: ptr.String("Scope"),
			},
		},
	}
}
