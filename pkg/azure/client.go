package azure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
)

type Client interface {
	RegisterOrUpdateApplication(credential v1alpha1.AzureAdCredential) (Credentials, error)
	DeleteApplication(credential v1alpha1.AzureAdCredential) error
}

type client struct {
	ctx                    context.Context
	config                 *Config
	servicePrincipalClient graphrbac.ServicePrincipalsClient
	applicationsClient     graphrbac.ApplicationsClient
}

type Credentials struct {
	Public  Public  `json:"public"`
	Private Private `json:"private"`
}

type Public struct {
	ClientId string `json:"clientId"`
	Key      Key    `json:"key"`
}

type Private struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Key          Key    `json:"key"`
}

type Key struct {
	KeyBase64 string `json:"keyBase64"`
	KeyId     string `json:"keyId"`
}

const (
	IntegratedAppTag string = "WindowsAzureActiveDirectoryIntegratedApp"
	SignInAudience   string = "AzureADMyOrg"
)

func NewClient(ctx context.Context, cfg *Config) (Client, error) {
	spClient, err := getServicePrincipalsClient(cfg)
	if err != nil {
		return nil, err
	}

	appClient, err := getApplicationsClient(cfg)
	if err != nil {
		return nil, err
	}

	return newClient(ctx, cfg, spClient, appClient), nil
}

// RegisterOrUpdateApplication registers an AAD application if it does not exist, otherwise updates the existing application.
func (c client) RegisterOrUpdateApplication(credential v1alpha1.AzureAdCredential) (Credentials, error) {
	exists, err := c.applicationExists(credential)
	if err != nil {
		return Credentials{}, err
	}
	if exists {
		return c.updateApplication(credential)
	} else {
		return c.registerApplication(credential)
	}
}

// DeleteApplication deletes the specified AAD application.
func (c client) DeleteApplication(credential v1alpha1.AzureAdCredential) error {
	// TODO
	return nil
}

func getServicePrincipalsClient(cfg *Config) (graphrbac.ServicePrincipalsClient, error) {
	spClient := graphrbac.NewServicePrincipalsClient(cfg.Tenant)
	a, err := GetGraphAuthorizer(cfg)
	if err != nil {
		return spClient, err
	}
	spClient.Authorizer = a
	return spClient, nil
}

func getApplicationsClient(cfg *Config) (graphrbac.ApplicationsClient, error) {
	appClient := graphrbac.NewApplicationsClient(cfg.Tenant)
	a, err := GetGraphAuthorizer(cfg)
	if err != nil {
		return appClient, err
	}
	appClient.Authorizer = a
	return appClient, nil
}

func newClient(ctx context.Context, cfg *Config, spClient graphrbac.ServicePrincipalsClient, appClient graphrbac.ApplicationsClient) Client {
	return client{
		ctx:                    ctx,
		config:                 cfg,
		servicePrincipalClient: spClient,
		applicationsClient:     appClient,
	}
}

func (c client) allApplications(filters ...string) ([]graphrbac.Application, error) {
	var applications []graphrbac.Application
	var result graphrbac.ApplicationListResultPage

	result, err := c.applicationsClient.List(c.ctx, mapFiltersToFilter(filters))
	if err != nil {
		return nil, err
	}
	for {
		applications = append(applications, result.Values()...)
		err = result.NextWithContext(c.ctx)
		if err != nil {
			return nil, err
		}
		if !result.NotDone() {
			return applications, nil
		}
	}
}

// TODO
func (c client) addClientSecret(credential v1alpha1.AzureAdCredential) {
	_, _ = c.applicationsClient.UpdatePasswordCredentials(c.ctx, "", graphrbac.PasswordCredentialsUpdateParameters{
		Value: &[]graphrbac.PasswordCredential{
			{
				StartDate: &date.Time{Time: time.Now()},
				EndDate:   &date.Time{Time: time.Now().AddDate(0, 0, 1)},
				KeyID:     to.StringPtr("mykeyid"),
				Value:     to.StringPtr("mypassword"),
			},
		},
	})
}

func (c client) applicationExists(credential v1alpha1.AzureAdCredential) (bool, error) {
	applications, err := c.allApplications(filterByName(credential.GetName()))
	if err != nil {
		return false, err
	}
	return len(applications) > 0, nil
}

func mapFiltersToFilter(filters []string) string {
	if len(filters) > 0 {
		return strings.Join(filters[:], " ")
	} else {
		return ""
	}
}

func filterByName(name string) string {
	return fmt.Sprintf("displayName eq '%s'", name)
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

func (c client) registerApplication(credential v1alpha1.AzureAdCredential) (Credentials, error) {
	application, err := c.applicationsClient.Create(c.ctx, applicationCreateParameters(credential))
	if err != nil {
		return Credentials{}, err

	}
	return Credentials{
		Public: Public{
			ClientId: *application.AppID,
			Key: Key{
				KeyBase64: "",
				KeyId:     "",
			},
		},
		Private: Private{
			ClientId:     *application.AppID,
			ClientSecret: "",
			Key: Key{
				KeyBase64: "",
				KeyId:     "",
			},
		},
	}, nil
}

// TODO
func (c client) updateApplication(credential v1alpha1.AzureAdCredential) (Credentials, error) {
	return Credentials{}, nil
}
