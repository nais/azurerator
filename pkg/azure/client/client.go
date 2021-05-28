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
	"github.com/nais/azureator/pkg/config"
)

const (
	MaxNumberOfPagesToFetch           = 1000
	DelayIntervalBetweenModifications = 3 * time.Second
	DelayIntervalBetweenCreations     = 5 * time.Second
)

type client struct {
	config      *config.AzureConfig
	httpClient  *http.Client
	graphClient *msgraph.GraphServiceRequestBuilder
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

	return client{
		config:      cfg,
		httpClient:  httpClient,
		graphClient: graphClient,
	}, nil
}

// Create registers a new AAD application with the desired configuration
func (c client) Create(tx azure.Transaction) (*azure.ApplicationResult, error) {
	app, err := c.application().register(tx)
	if err != nil {
		return nil, fmt.Errorf("registering application resource: %w", err)
	}

	tx = tx.UpdateWithApplicationIDs(*app)

	// sleep to allow replication across Microsoft's systems...
	time.Sleep(DelayIntervalBetweenCreations)

	servicePrincipal, err := c.servicePrincipal().register(tx)
	if err != nil {
		return nil, fmt.Errorf("registering service principal for application: %w", err)
	}

	tx = tx.UpdateWithServicePrincipalID(servicePrincipal)

	time.Sleep(DelayIntervalBetweenCreations)

	if err := c.application().identifierUri().set(tx); err != nil {
		return nil, fmt.Errorf("setting identifier URIs for application: %w", err)
	}

	preAuthApps, err := c.process(tx)
	if err != nil {
		return nil, err
	}

	return &azure.ApplicationResult{
		ClientId:           *app.AppID,
		ObjectId:           *app.ID,
		ServicePrincipalId: *servicePrincipal.ID,
		PreAuthorizedApps:  *preAuthApps,
		Tenant:             c.config.Tenant.Id,
		Result:             azure.OperationResultCreated,
	}, nil
}

// Delete deletes the specified AAD application.
func (c client) Delete(tx azure.Transaction) error {
	_, exists, err := c.Exists(tx)
	if err != nil {
		return err
	}
	if exists {
		return c.application().delete(tx)
	}
	return fmt.Errorf("application does not exist: %s (clientId: %s, objectId: %s)", kubernetes.UniformResourceName(&tx.Instance), tx.Instance.GetClientId(), tx.Instance.GetObjectId())
}

// Exists returns an indication of whether the application exists in AAD or not
func (c client) Exists(tx azure.Transaction) (*msgraph.Application, bool, error) {
	return c.application().exists(tx)
}

// Get returns a Graph API Application entity, which represents an Application in AAD
func (c client) Get(tx azure.Transaction) (msgraph.Application, error) {
	return c.application().getByName(tx.Ctx, kubernetes.UniformResourceName(&tx.Instance))
}

// GetServicePrincipal returns the application's associated Graph ServicePrincipal entity, or registers and returns one if none exist for the application.
func (c client) GetServicePrincipal(tx azure.Transaction) (msgraph.ServicePrincipal, error) {
	clientId := tx.Instance.GetClientId()
	exists, sp, err := c.servicePrincipal().exists(tx.Ctx, clientId)
	if err != nil {
		return msgraph.ServicePrincipal{}, fmt.Errorf("looking up existence of service principal: %w", err)
	}
	if exists {
		return sp, nil
	}
	sp, err = c.servicePrincipal().register(tx)
	if err != nil {
		return msgraph.ServicePrincipal{}, fmt.Errorf("registering service principal that did not exist: %w", err)
	}
	return sp, nil
}

// GetPreAuthorizedApps transforms a list of desired pre-authorized applications in the spec to lists of valid and invalid
// Azure applications, where the validity indicates whether a desired application is pre-authorized or not.
func (c client) GetPreAuthorizedApps(tx azure.Transaction) (*azure.PreAuthorizedApps, error) {
	return c.preAuthApps().get(tx)
}

// AddCredentials adds credentials for an existing AAD application
func (c client) AddCredentials(tx azure.Transaction) (azure.CredentialsSet, error) {
	// sleep to prevent concurrent modification error from Microsoft
	time.Sleep(DelayIntervalBetweenModifications)

	currPasswordCredential, err := c.passwordCredential().add(tx)
	if err != nil {
		return azure.CredentialsSet{}, fmt.Errorf("adding current password credential: %w", err)
	}

	time.Sleep(DelayIntervalBetweenModifications)

	nextPasswordCredential, err := c.passwordCredential().add(tx)
	if err != nil {
		return azure.CredentialsSet{}, fmt.Errorf("adding next password credential: %w", err)
	}

	time.Sleep(DelayIntervalBetweenModifications)

	keyCredentialSet, err := c.keyCredential().add(tx)
	if err != nil {
		return azure.CredentialsSet{}, fmt.Errorf("adding key credential set: %w", err)
	}

	return azure.CredentialsSet{
		Current: azure.Credentials{
			Certificate: azure.Certificate{
				KeyId: string(*keyCredentialSet.Current.KeyCredential.KeyID),
				Jwk:   keyCredentialSet.Current.Jwk,
			},
			Password: azure.Password{
				KeyId:        string(*currPasswordCredential.KeyID),
				ClientSecret: *currPasswordCredential.SecretText,
			},
		},
		Next: azure.Credentials{
			Certificate: azure.Certificate{
				KeyId: string(*keyCredentialSet.Next.KeyCredential.KeyID),
				Jwk:   keyCredentialSet.Next.Jwk,
			},
			Password: azure.Password{
				KeyId:        string(*nextPasswordCredential.KeyID),
				ClientSecret: *nextPasswordCredential.SecretText,
			},
		},
	}, nil
}

// RotateCredentials rotates credentials for an existing AAD application
func (c client) RotateCredentials(tx azure.Transaction, existing azure.CredentialsSet, inUse azure.KeyIdsInUse) (azure.CredentialsSet, error) {
	time.Sleep(DelayIntervalBetweenModifications) // sleep to prevent concurrent modification error from Microsoft

	nextPasswordCredential, err := c.passwordCredential().rotate(tx, existing, inUse)
	if err != nil {
		return azure.CredentialsSet{}, fmt.Errorf("rotating password credential: %w", err)
	}

	time.Sleep(DelayIntervalBetweenModifications)

	nextKeyCredential, nextJwk, err := c.keyCredential().rotate(tx, existing, inUse)
	if err != nil {
		return azure.CredentialsSet{}, fmt.Errorf("rotating key credential: %w", err)
	}

	return azure.CredentialsSet{
		Current: existing.Next,
		Next: azure.Credentials{
			Certificate: azure.Certificate{
				KeyId: string(*nextKeyCredential.KeyID),
				Jwk:   *nextJwk,
			},
			Password: azure.Password{
				KeyId:        string(*nextPasswordCredential.KeyID),
				ClientSecret: *nextPasswordCredential.SecretText,
			},
		},
	}, nil
}

// ValidateCredentials validates the given credentials set against the actual state for the application in Azure AD.
func (c client) ValidateCredentials(tx azure.Transaction, existing azure.CredentialsSet) (bool, error) {
	validPasswordCredentials, err := c.passwordCredential().validate(tx, existing)
	if err != nil {
		return false, fmt.Errorf("validating password credentials: %w", err)
	}

	validateKeyCredentials, err := c.keyCredential().validate(tx, existing)
	if err != nil {
		return false, fmt.Errorf("validating key credentials: %w", err)
	}

	return validPasswordCredentials && validateKeyCredentials, nil
}

// Update updates an existing AAD application. Should be an idempotent operation
func (c client) Update(tx azure.Transaction) (*azure.ApplicationResult, error) {
	clientId := tx.Instance.GetClientId()
	objectId := tx.Instance.GetObjectId()
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	if err := c.application().update(tx); err != nil {
		return nil, fmt.Errorf("updating application resource: %w", err)
	}

	if err := c.application().redirectUri().update(tx); err != nil {
		return nil, fmt.Errorf("updating redirect URIs: %w", err)
	}

	preAuthApps, err := c.process(tx)
	if err != nil {
		return nil, err
	}

	return &azure.ApplicationResult{
		ClientId:           clientId,
		ObjectId:           objectId,
		ServicePrincipalId: servicePrincipalId,
		PreAuthorizedApps:  *preAuthApps,
		Tenant:             c.config.Tenant.Id,
		Result:             azure.OperationResultUpdated,
	}, nil
}

func (c client) process(tx azure.Transaction) (*azure.PreAuthorizedApps, error) {
	if err := c.oAuth2PermissionGrant().process(tx); err != nil {
		return nil, fmt.Errorf("processing oauth2 permission grants: %w", err)
	}

	preAuthApps, err := c.preAuthApps().process(tx)
	if err != nil {
		return nil, fmt.Errorf("processing preauthorized apps: %w", err)
	}

	if c.config.Features.TeamsManagement.Enabled {
		if err = c.teamowners().process(tx); err != nil {
			return nil, fmt.Errorf("processing owners: %w", err)
		}
	}

	if c.config.Features.ClaimsMappingPolicies.Enabled {
		if err := c.servicePrincipal().policies().process(tx); err != nil {
			return nil, fmt.Errorf("processing service principal policies: %w", err)
		}
	}

	if c.config.Features.GroupsAssignment.Enabled {
		if err := c.groups().process(tx); err != nil {
			return nil, fmt.Errorf("processing groups to service principal: %w", err)
		}
	}

	if c.config.Features.AppRoleAssignmentRequired.Enabled {
		if err := c.servicePrincipal().setAppRoleAssignmentRequired(tx); err != nil {
			return nil, fmt.Errorf("enabling requirement for approle assignments: %w", err)
		}
	}

	return preAuthApps, nil
}
