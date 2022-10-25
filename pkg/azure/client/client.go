package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nais/msgraph.go/msauth"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"golang.org/x/oauth2"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application"
	"github.com/nais/azureator/pkg/azure/client/application/identifieruri"
	"github.com/nais/azureator/pkg/azure/client/group"
	"github.com/nais/azureator/pkg/azure/client/oauth2permissiongrant"
	"github.com/nais/azureator/pkg/azure/client/preauthorizedapp"
	"github.com/nais/azureator/pkg/azure/client/serviceprincipal"
	"github.com/nais/azureator/pkg/azure/permissions"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/retry"
	"github.com/nais/azureator/pkg/transaction"
)

const (
	RetryInitialDelay    = 1 * time.Second
	RetryMaximumDuration = 30 * time.Second
)

type Client struct {
	config      *config.AzureConfig
	httpClient  *http.Client
	graphClient *msgraph.GraphServiceRequestBuilder
}

func (c Client) Config() *config.AzureConfig {
	return c.config
}

func (c Client) HttpClient() *http.Client {
	return c.httpClient
}

func (c Client) GraphClient() *msgraph.GraphServiceRequestBuilder {
	return c.graphClient
}

func (c Client) MaxNumberOfPagesToFetch() int {
	return c.config.Pagination.MaxPages
}

func (c Client) DelayIntervalBetweenModifications() time.Duration {
	return c.config.Delay.BetweenModifications
}

func (c Client) Application() application.Application {
	return application.NewApplication(c)
}

func (c Client) AppRoleAssignments(tx transaction.Transaction, targetId azure.ServicePrincipalId) serviceprincipal.AppRoleAssignments {
	return serviceprincipal.NewAppRoleAssignments(c, tx, targetId)
}

func (c Client) Credentials() azure.Credentials {
	return NewCredentials(c)
}

func (c Client) Groups() group.Groups {
	return group.NewGroup(c)
}

func (c Client) OAuth2PermissionGrant() oauth2permissiongrant.OAuth2PermissionGrant {
	return oauth2permissiongrant.NewOAuth2PermissionGrant(c)
}

func (c Client) PreAuthApps() preauthorizedapp.PreAuthApps {
	return preauthorizedapp.NewPreAuthApps(c)
}

func (c Client) ServicePrincipal() serviceprincipal.ServicePrincipal {
	return serviceprincipal.NewServicePrincipal(c)
}

func New(ctx context.Context, cfg *config.AzureConfig) (azure.Client, error) {
	m := msauth.NewManager()
	scopes := []string{msauth.DefaultMSGraphScope}
	ts, err := m.ClientCredentialsGrant(ctx, cfg.Tenant.Id, cfg.Auth.ClientId, cfg.Auth.ClientSecret, scopes)
	if err != nil {
		return nil, fmt.Errorf("instantiating graph client: %w", err)
	}

	httpClient := oauth2.NewClient(ctx, ts)
	graphClient := msgraph.NewClient(httpClient)

	return Client{
		config:      cfg,
		httpClient:  httpClient,
		graphClient: graphClient,
	}, nil
}

// Create registers a new AAD application with the desired configuration
func (c Client) Create(tx transaction.Transaction) (*result.Application, error) {
	app, err := c.Application().Register(tx)
	if err != nil {
		return nil, fmt.Errorf("registering application resource: %w", err)
	}

	tx = tx.UpdateWithApplicationIDs(*app)

	var servicePrincipal msgraph.ServicePrincipal
	err = doRetry(tx.Ctx, func(ctx context.Context) error {
		servicePrincipal, err = c.ServicePrincipal().Register(tx)
		return retry.RetryableError(err)
	})
	if err != nil {
		return nil, fmt.Errorf("registering service principal for application: %w", err)
	}

	tx = tx.UpdateWithServicePrincipalID(servicePrincipal)

	identifierUris := identifieruri.DescribeCreate(tx.Instance, tx.ClusterName)
	err = doRetry(tx.Ctx, func(ctx context.Context) error {
		err := c.Application().IdentifierUri().Set(tx, identifierUris)
		return retry.RetryableError(err)
	})
	if err != nil {
		return nil, fmt.Errorf("setting identifier URIs for application: %w", err)
	}

	actualPermissions := permissions.ExtractPermissions(app)

	var preAuthApps *result.PreAuthorizedApps
	err = doRetry(tx.Ctx, func(ctx context.Context) error {
		preAuthApps, err = c.process(tx, actualPermissions)
		return retry.RetryableError(err)
	})
	if err != nil {
		return nil, err
	}

	return &result.Application{
		ClientId:           *app.AppID,
		ObjectId:           *app.ID,
		ServicePrincipalId: *servicePrincipal.ID,
		Permissions:        actualPermissions,
		PreAuthorizedApps:  *preAuthApps,
		Tenant:             c.config.Tenant.Id,
		Result:             result.OperationCreated,
	}, nil
}

// Delete deletes the specified AAD application.
func (c Client) Delete(tx transaction.Transaction) error {
	_, exists, err := c.Exists(tx)
	if err != nil {
		return err
	}
	if exists {
		return c.Application().Delete(tx)
	}

	return fmt.Errorf("application does not exist: %s (clientId: %s, objectId: %s)", tx.UniformResourceName, tx.Instance.GetClientId(), tx.Instance.GetObjectId())
}

// Exists returns an indication of whether the application exists in AAD or not
func (c Client) Exists(tx transaction.Transaction) (*msgraph.Application, bool, error) {
	return c.Application().Exists(tx)
}

// Get returns a Graph API Application entity, which represents an Application in AAD
func (c Client) Get(tx transaction.Transaction) (msgraph.Application, error) {
	return c.Application().Get(tx)
}

// GetServicePrincipal returns the application's associated Graph ServicePrincipal entity, or registers and returns one if none exist for the application.
func (c Client) GetServicePrincipal(tx transaction.Transaction) (msgraph.ServicePrincipal, error) {
	clientId := tx.Instance.GetClientId()
	exists, sp, err := c.ServicePrincipal().Exists(tx.Ctx, clientId)
	if err != nil {
		return msgraph.ServicePrincipal{}, fmt.Errorf("looking up existence of service principal: %w", err)
	}
	if exists {
		return sp, nil
	}
	sp, err = c.ServicePrincipal().Register(tx)
	if err != nil {
		return msgraph.ServicePrincipal{}, fmt.Errorf("registering service principal that did not exist: %w", err)
	}
	return sp, nil
}

// GetPreAuthorizedApps transforms a list of desired pre-authorized applications in the spec to lists of valid and invalid
// Azure applications, where the validity indicates whether a desired application is pre-authorized or not.
func (c Client) GetPreAuthorizedApps(tx transaction.Transaction) (*result.PreAuthorizedApps, error) {
	return c.PreAuthApps().Get(tx)
}

// Update updates an existing AAD application. Should be an idempotent operation
func (c Client) Update(tx transaction.Transaction) (*result.Application, error) {
	clientId := tx.Instance.GetClientId()
	objectId := tx.Instance.GetObjectId()
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	app, err := c.Application().Update(tx)
	if err != nil {
		return nil, fmt.Errorf("updating application resource: %w", err)
	}

	if err := c.Application().RedirectUri().Update(tx); err != nil {
		return nil, fmt.Errorf("updating redirect URIs: %w", err)
	}

	actualPermissions := permissions.ExtractPermissions(app)
	preAuthApps, err := c.process(tx, actualPermissions)
	if err != nil {
		return nil, err
	}

	if err := c.Application().RemoveDisabledPermissions(tx, *app); err != nil {
		return nil, err
	}

	return &result.Application{
		ClientId:           clientId,
		ObjectId:           objectId,
		ServicePrincipalId: servicePrincipalId,
		Permissions:        actualPermissions,
		PreAuthorizedApps:  *preAuthApps,
		Tenant:             c.config.Tenant.Id,
		Result:             result.OperationUpdated,
	}, nil
}

func (c Client) process(tx transaction.Transaction, permissions permissions.Permissions) (*result.PreAuthorizedApps, error) {
	if err := c.OAuth2PermissionGrant().Process(tx); err != nil {
		return nil, fmt.Errorf("processing oauth2 permission grants: %w", err)
	}

	preAuthApps, err := c.PreAuthApps().Process(tx, permissions)
	if err != nil {
		return nil, fmt.Errorf("processing preauthorized apps: %w", err)
	}

	if c.config.Features.ClaimsMappingPolicies.Enabled {
		if err := c.ServicePrincipal().Policies().Process(tx); err != nil {
			return nil, fmt.Errorf("processing service principal policies: %w", err)
		}
	}

	if c.config.Features.GroupsAssignment.Enabled {
		if err := c.Groups().Process(tx); err != nil {
			return nil, fmt.Errorf("processing groups to service principal: %w", err)
		}
	}

	if c.config.Features.AppRoleAssignmentRequired.Enabled {
		if err := c.ServicePrincipal().SetAppRoleAssignmentRequired(tx); err != nil {
			return nil, fmt.Errorf("enabling requirement for approle assignments: %w", err)
		}
	}

	return preAuthApps, nil
}

func doRetry(ctx context.Context, fn func(context.Context) error) error {
	return retry.Fibonacci(RetryInitialDelay).
		WithMaxDuration(RetryMaximumDuration).
		Do(ctx, fn)
}
