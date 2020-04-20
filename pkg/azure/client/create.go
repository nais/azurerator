package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Create registers a new AAD application
func (c client) Create(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	return c.registerApplication(ctx, credential)
}

func (c client) registerApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	application, err := c.graphClient.Applications().Request().Add(ctx, applicationCreateParameters(credential))
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to register application: %w", err)
	}
	clientSecret, err := c.addClientSecret(ctx, *application.ID)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to update credentials for application %w", err)
	}

	return azure.Application{
		Credentials: azure.Credentials{
			Public: azure.Public{
				ClientId: *application.AppID,
				Key: azure.Key{
					Base64: "",
				},
			},
			Private: azure.Private{
				ClientId:     *application.AppID,
				ClientSecret: *clientSecret.SecretText,
				Key: azure.Key{
					Base64: "",
				},
			},
		},
		ClientId:         *application.AppID,
		ObjectId:         *application.ID,
		PasswordKeyId:    string(*clientSecret.KeyID),
		CertificateKeyId: "",
	}, nil
}

// TODO
func applicationCreateParameters(credential v1alpha1.AzureAdCredential) *msgraph.Application {
	return &msgraph.Application{
		DisplayName:           ptr.String(credential.Name),
		IdentifierUris:        nil,
		AppRoles:              nil,
		GroupMembershipClaims: ptr.String(SecurityGroup),
		KeyCredentials:        nil,
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

func getReplyUrlsStringSlice(credential v1alpha1.AzureAdCredential) []string {
	var replyUrls []string
	for _, v := range credential.Spec.ReplyUrls {
		replyUrls = append(replyUrls, v.Url)
	}
	return replyUrls
}
