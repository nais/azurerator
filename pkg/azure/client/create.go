package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"gopkg.in/square/go-jose.v2"
)

// Create registers a new AAD application
func (c client) Create(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	return c.registerApplication(ctx, credential)
}

func (c client) registerApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	jwkPair, err := crypto.GenerateJwkPair(credential)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to generate JWK pair for application: %w", err)
	}
	keyCredential := createKeyCredential(jwkPair.Public)

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

// TODO - fill in 'nil's or remove
func createApplication(credential v1alpha1.AzureAdCredential, keyCredential msgraph.KeyCredential) *msgraph.Application {
	return &msgraph.Application{
		DisplayName:           ptr.String(credential.GetUniqueName()),
		IdentifierUris:        nil,
		AppRoles:              nil,
		GroupMembershipClaims: ptr.String(SecurityGroup),
		KeyCredentials:        []msgraph.KeyCredential{keyCredential},
		OptionalClaims:        nil,
		Web: &msgraph.WebApplication{
			RedirectUris: getReplyUrlsStringSlice(credential),
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
	}
}

func createKeyCredential(jwk jose.JSONWebKey) msgraph.KeyCredential {
	keyId := msgraph.UUID(uuid.New().String())
	keyBase64 := msgraph.Binary(crypto.ConvertToPem(jwk.Certificates[0]))
	return msgraph.KeyCredential{
		KeyID:       &keyId,
		DisplayName: ptr.String("azurerator"),
		Type:        ptr.String("AsymmetricX509Cert"),
		Usage:       ptr.String("Verify"),
		Key:         &keyBase64,
	}
}

func getReplyUrlsStringSlice(credential v1alpha1.AzureAdCredential) []string {
	var replyUrls []string
	for _, v := range credential.Spec.ReplyUrls {
		replyUrls = append(replyUrls, v.Url)
	}
	return replyUrls
}
