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

const (
	IntegratedAppTag string = "WindowsAzureActiveDirectoryIntegratedApp"
	SignInAudience   string = "AzureADMyOrg"
)

func NewClient(ctx context.Context, cfg *Config) (Client, error) {
	spClient, err := getServicePrincipalsClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate service principal client: %w", err)
	}

	appClient, err := getApplicationsClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate applications client: %w", err)
	}

	return newClient(ctx, cfg, spClient, appClient), nil
}

// ApplicationExists returns an indication of whether the application exists in AAD or not
func (c client) ApplicationExists(credential v1alpha1.AzureAdCredential) (bool, error) {
	exists, err := c.applicationExists(credential)
	if err != nil {
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	return exists, nil
}

// RegisterApplication registers a new AAD application
func (c client) RegisterApplication(credential v1alpha1.AzureAdCredential) (Application, error) {
	return c.registerApplication(credential)
}

// UpdateApplication updates an existing AAD application
func (c client) UpdateApplication(credential v1alpha1.AzureAdCredential) (Application, error) {
	return c.updateApplication(credential)
}

// DeleteApplication deletes the specified AAD application.
func (c client) DeleteApplication(credential v1alpha1.AzureAdCredential) error {
	exists, err := c.applicationExists(credential)
	if err != nil {
		return err
	}
	if exists {
		return c.deleteApplication(credential)
	}
	return fmt.Errorf("application does not exist: %s (clientId: %s, objectId: %s)", credential.Name, credential.Status.ClientId, credential.Status.ObjectId)
}

func getServicePrincipalsClient(cfg *Config) (graphrbac.ServicePrincipalsClient, error) {
	spClient := graphrbac.NewServicePrincipalsClient(cfg.Tenant)
	a, err := GetGraphAuthorizer(cfg)
	if err != nil {
		return spClient, fmt.Errorf("failed to get graph authorizer: %w", err)
	}
	spClient.Authorizer = a
	return spClient, nil
}

func getApplicationsClient(cfg *Config) (graphrbac.ApplicationsClient, error) {
	appClient := graphrbac.NewApplicationsClient(cfg.Tenant)
	a, err := GetGraphAuthorizer(cfg)
	if err != nil {
		return appClient, fmt.Errorf("failed to get graph authorizer: %w", err)
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
		return nil, fmt.Errorf("failed to get list applications: %w", err)
	}
	for {
		applications = append(applications, result.Values()...)
		err = result.NextWithContext(c.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get list applications: %w", err)
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
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
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

func (c client) registerApplication(credential v1alpha1.AzureAdCredential) (Application, error) {
	application, err := c.applicationsClient.Create(c.ctx, applicationCreateParameters(credential))
	if err != nil {
		return Application{}, fmt.Errorf("failed to register application: %w", err)
	}
	return Application{
		Credentials: Credentials{
			Public: Public{
				ClientId: *application.AppID,
				Key: Key{
					Base64: "",
				},
			},
			Private: Private{
				ClientId:     *application.AppID,
				ClientSecret: "",
				Key: Key{
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

func (c client) deleteApplication(credential v1alpha1.AzureAdCredential) error {
	if _, err := c.applicationsClient.Delete(c.ctx, credential.Status.ObjectId); err != nil {
		return fmt.Errorf("failed delete application: %w", err)
	}
	return nil
}

// TODO
func (c client) updateApplication(credential v1alpha1.AzureAdCredential) (Application, error) {
	return Application{}, nil
}
