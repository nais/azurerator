package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
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
	}
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
