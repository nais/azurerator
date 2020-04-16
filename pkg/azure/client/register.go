package client

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
)

// RegisterApplication registers a new AAD application
func (c client) RegisterApplication(credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	return c.registerApplication(credential)
}

func (c client) registerApplication(credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	application, err := c.applicationsClient.Create(c.ctx, applicationCreateParameters(credential))
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to register application: %w", err)
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
				ClientSecret: "",
				Key: azure.Key{
					Base64: "",
				},
			},
		},
		ClientId:         *application.AppID,
		ObjectId:         *application.ObjectID,
		PasswordKeyId:    "",
		CertificateKeyId: "",
	}, nil
}

// TODO
func applicationCreateParameters(credential v1alpha1.AzureAdCredential) graphrbac.ApplicationCreateParameters {
	return graphrbac.ApplicationCreateParameters{
		DisplayName:                to.StringPtr(credential.Name),
		IdentifierUris:             nil,
		AppLogoURL:                 nil,
		AppRoles:                   nil,
		AppPermissions:             nil,
		AvailableToOtherTenants:    to.BoolPtr(false),
		ErrorURL:                   nil,
		GroupMembershipClaims:      graphrbac.SecurityGroup,
		Homepage:                   nil,
		InformationalUrls:          nil,
		IsDeviceOnlyAuthSupported:  nil,
		KeyCredentials:             nil,
		KnownClientApplications:    nil,
		LogoutURL:                  nil,
		Oauth2AllowImplicitFlow:    to.BoolPtr(false),
		Oauth2AllowURLPathMatching: nil,
		Oauth2Permissions:          nil,
		Oauth2RequirePostResponse:  nil,
		OrgRestrictions:            nil,
		OptionalClaims:             nil,
		PasswordCredentials:        nil,
		ReplyUrls:                  to.StringSlicePtr(getReplyUrlsStringSlice(credential)),
		SignInAudience:             nil,
	}
}

func getReplyUrlsStringSlice(credential v1alpha1.AzureAdCredential) []string {
	var replyUrls []string
	for _, v := range credential.Spec.ReplyUrls {
		replyUrls = append(replyUrls, v.Url)
	}
	return replyUrls
}
