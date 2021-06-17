package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/nais/msgraph.go/msauth"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"golang.org/x/oauth2"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application"
	"github.com/nais/azureator/pkg/azure/client/approleassignment"
	"github.com/nais/azureator/pkg/azure/client/group"
	"github.com/nais/azureator/pkg/azure/client/keycredential"
	"github.com/nais/azureator/pkg/azure/client/oauth2permissiongrant"
	"github.com/nais/azureator/pkg/azure/client/passwordcredential"
	"github.com/nais/azureator/pkg/azure/client/preauthorizedapp"
	"github.com/nais/azureator/pkg/azure/client/serviceprincipal"
	"github.com/nais/azureator/pkg/azure/client/team"
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/permissions"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/azure/transaction"
	"github.com/nais/azureator/pkg/config"
)

const (
	maxNumberOfPagesToFetch           = 1000
	delayIntervalBetweenModifications = 3 * time.Second
	delayIntervalBetweenCreations     = 5 * time.Second
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
	return maxNumberOfPagesToFetch
}

func (c Client) DelayIntervalBetweenModifications() time.Duration {
	return delayIntervalBetweenModifications
}

func (c Client) DelayIntervalBetweenCreations() time.Duration {
	return delayIntervalBetweenCreations
}

func (c Client) Application() azure.Application {
	return application.NewApplication(c)
}

func (c Client) AppRoleAssignments(roleId msgraph.UUID, targetId azure.ObjectId) azure.AppRoleAssignmentsWithRoleId {
	return approleassignment.NewAppRoleAssignmentsWithRoleId(c, c.AppRoleAssignmentsNoRoleId(targetId), roleId)
}

func (c Client) AppRoleAssignmentsNoRoleId(targetId azure.ObjectId) azure.AppRoleAssignments {
	return approleassignment.NewAppRoleAssignmentsNoRoleId(c, targetId)
}

func (c Client) Groups() azure.Groups {
	return group.NewGroup(c)
}

func (c Client) KeyCredential() azure.KeyCredential {
	return keycredential.NewKeyCredential(c)
}

func (c Client) OAuth2PermissionGrant() azure.OAuth2PermissionGrant {
	return oauth2permissiongrant.NewOAuth2PermissionGrant(c)
}

func (c Client) PasswordCredential() azure.PasswordCredential {
	return passwordcredential.NewPasswordCredential(c)
}

func (c Client) PreAuthApps() azure.PreAuthApps {
	return preauthorizedapp.NewPreAuthApps(c)
}

func (c Client) ServicePrincipal() azure.ServicePrincipal {
	return serviceprincipal.NewServicePrincipal(c)
}

func (c Client) Team() azure.Team {
	return team.NewTeam(c)
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

	// sleep to allow replication across Microsoft's systems...
	time.Sleep(delayIntervalBetweenCreations)

	servicePrincipal, err := c.ServicePrincipal().Register(tx)
	if err != nil {
		return nil, fmt.Errorf("registering service principal for application: %w", err)
	}

	tx = tx.UpdateWithServicePrincipalID(servicePrincipal)

	time.Sleep(delayIntervalBetweenCreations)

	if err := c.Application().IdentifierUri().Set(tx); err != nil {
		return nil, fmt.Errorf("setting identifier URIs for application: %w", err)
	}

	actualPermissions := permissions.ExtractPermissions(app)
	preAuthApps, err := c.process(tx, actualPermissions)
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
	return fmt.Errorf("application does not exist: %s (clientId: %s, objectId: %s)", kubernetes.UniformResourceName(&tx.Instance), tx.Instance.GetClientId(), tx.Instance.GetObjectId())
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

// AddCredentials adds credentials for an existing AAD application
func (c Client) AddCredentials(tx transaction.Transaction) (credentials.Set, error) {
	// sleep to prevent concurrent modification error from Microsoft
	time.Sleep(delayIntervalBetweenModifications)

	currPasswordCredential, err := c.PasswordCredential().Add(tx)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("adding current password credential: %w", err)
	}

	time.Sleep(delayIntervalBetweenModifications)

	nextPasswordCredential, err := c.PasswordCredential().Add(tx)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("adding next password credential: %w", err)
	}

	time.Sleep(delayIntervalBetweenModifications)

	keyCredentialSet, err := c.KeyCredential().Add(tx)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("adding key credential set: %w", err)
	}

	return credentials.Set{
		Current: credentials.Credentials{
			Certificate: credentials.Certificate{
				KeyId: string(*keyCredentialSet.Current.KeyCredential.KeyID),
				Jwk:   keyCredentialSet.Current.Jwk,
			},
			Password: credentials.Password{
				KeyId:        string(*currPasswordCredential.KeyID),
				ClientSecret: *currPasswordCredential.SecretText,
			},
		},
		Next: credentials.Credentials{
			Certificate: credentials.Certificate{
				KeyId: string(*keyCredentialSet.Next.KeyCredential.KeyID),
				Jwk:   keyCredentialSet.Next.Jwk,
			},
			Password: credentials.Password{
				KeyId:        string(*nextPasswordCredential.KeyID),
				ClientSecret: *nextPasswordCredential.SecretText,
			},
		},
	}, nil
}

// RotateCredentials rotates credentials for an existing AAD application
func (c Client) RotateCredentials(tx transaction.Transaction, existing credentials.Set, inUse credentials.KeyIdsInUse) (credentials.Set, error) {
	time.Sleep(delayIntervalBetweenModifications) // sleep to prevent concurrent modification error from Microsoft

	nextPasswordCredential, err := c.PasswordCredential().Rotate(tx, existing, inUse)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("rotating password credential: %w", err)
	}

	time.Sleep(delayIntervalBetweenModifications)

	nextKeyCredential, nextJwk, err := c.KeyCredential().Rotate(tx, existing, inUse)
	if err != nil {
		return credentials.Set{}, fmt.Errorf("rotating key credential: %w", err)
	}

	return credentials.Set{
		Current: existing.Next,
		Next: credentials.Credentials{
			Certificate: credentials.Certificate{
				KeyId: string(*nextKeyCredential.KeyID),
				Jwk:   *nextJwk,
			},
			Password: credentials.Password{
				KeyId:        string(*nextPasswordCredential.KeyID),
				ClientSecret: *nextPasswordCredential.SecretText,
			},
		},
	}, nil
}

// PurgeCredentials removes all credentials for the application in Azure AD.
func (c Client) PurgeCredentials(tx transaction.Transaction) error {
	err := c.PasswordCredential().Purge(tx)
	if err != nil {
		return fmt.Errorf("purging password credentials: %w", err)
	}

	err = c.KeyCredential().Purge(tx)
	if err != nil {
		return fmt.Errorf("purging key credentials: %w", err)
	}

	return nil
}

// ValidateCredentials validates the given credentials set against the actual state for the application in Azure AD.
func (c Client) ValidateCredentials(tx transaction.Transaction, existing credentials.Set) (bool, error) {
	validPasswordCredentials, err := c.PasswordCredential().Validate(tx, existing)
	if err != nil {
		return false, fmt.Errorf("validating password credentials: %w", err)
	}

	validateKeyCredentials, err := c.KeyCredential().Validate(tx, existing)
	if err != nil {
		return false, fmt.Errorf("validating key credentials: %w", err)
	}

	return validPasswordCredentials && validateKeyCredentials, nil
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

	// TODO: finally, patch application and remove all roles and scopes that are set to disabled
	//	- we _CANNOT delete a disabled PermissionScope that has been granted to any pre-authorized app
	// 	- we _CAN_ however delete a disabled AppRole _without_ removing the associated approleassignments first...
	//	(however it appears to clog up the list of granted Permissions in the Enterprise Apps overview)
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

	if c.config.Features.TeamsManagement.Enabled {
		if err = c.Team().Owners().Process(tx); err != nil {
			return nil, fmt.Errorf("processing owners: %w", err)
		}
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
