package client

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/liberator/pkg/kubernetes"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/msauth"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"golang.org/x/oauth2"
	"net/http"
)

const MaxNumberOfPagesToFetch = 1000

type client struct {
	config          *config.AzureConfig
	httpClient      *http.Client
	graphClient     *msgraph.GraphServiceRequestBuilder
	graphBetaClient *msgraphbeta.GraphServiceRequestBuilder
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
	graphBetaClient := msgraphbeta.NewClient(httpClient)

	return client{
		config:          cfg,
		httpClient:      httpClient,
		graphClient:     graphClient,
		graphBetaClient: graphBetaClient,
	}, nil
}

// Create registers a new AAD application with all the required accompanying resources
func (c client) Create(tx azure.Transaction) (*azure.ApplicationResult, error) {
	app := c.application()

	res, err := app.register(tx)
	if err != nil {
		return nil, fmt.Errorf("registering application resource: %w", err)
	}

	tx = tx.UpdateWithApplicationIDs(res.Application)

	servicePrincipal, err := c.servicePrincipal().register(tx)
	if err != nil {
		return nil, fmt.Errorf("registering service principal for application: %w", err)
	}

	tx = tx.UpdateWithServicePrincipalID(servicePrincipal)

	passwordCredential, err := c.passwordCredential().add(tx)
	if err != nil {
		return nil, fmt.Errorf("adding password credential: %w", err)
	}

	if err := app.identifierUri().set(tx); err != nil {
		return nil, fmt.Errorf("setting identifier URIs for application: %w", err)
	}

	preAuthApps, err := c.process(tx)
	if err != nil {
		return nil, err
	}

	lastPasswordKeyId := string(*passwordCredential.KeyID)
	lastCertificateKeyId := string(*res.KeyCredential.KeyID)

	return &azure.ApplicationResult{
		Certificate: azure.Certificate{
			KeyId: azure.KeyId{
				Latest:   lastCertificateKeyId,
				AllInUse: []string{lastCertificateKeyId},
			},
			Jwk: res.Jwk,
		},
		Password: azure.Password{
			KeyId: azure.KeyId{
				Latest:   lastPasswordKeyId,
				AllInUse: []string{lastPasswordKeyId},
			},
			ClientSecret: *passwordCredential.SecretText,
		},
		ClientId:           *res.Application.AppID,
		ObjectId:           *res.Application.ID,
		ServicePrincipalId: *servicePrincipal.ID,
		PreAuthorizedApps:  preAuthApps,
		Tenant:             c.config.Tenant.Id,
	}, nil
}

// Delete deletes the specified AAD application.
func (c client) Delete(tx azure.Transaction) error {
	exists, err := c.Exists(tx)
	if err != nil {
		return err
	}
	if exists {
		return c.application().delete(tx)
	}
	return fmt.Errorf("application does not exist: %s (clientId: %s, objectId: %s)", kubernetes.UniformResourceName(&tx.Instance), tx.Instance.GetClientId(), tx.Instance.GetObjectId())
}

// Exists returns an indication of whether the application exists in AAD or not
func (c client) Exists(tx azure.Transaction) (bool, error) {
	exists, err := c.application().exists(tx)
	if err != nil {
		return false, fmt.Errorf("looking up existence of application: %w", err)
	}
	return exists, nil
}

// Get returns a Graph API Application entity, which represents an Application in AAD
func (c client) Get(tx azure.Transaction) (msgraph.Application, error) {
	return c.application().getByName(tx.Ctx, kubernetes.UniformResourceName(&tx.Instance))
}

// GetServicePrincipal returns the application's associated Graph ServicePrincipal entity, or registers and returns one if none exist for the application.
func (c client) GetServicePrincipal(tx azure.Transaction) (msgraphbeta.ServicePrincipal, error) {
	clientId := tx.Instance.GetClientId()
	exists, sp, err := c.servicePrincipal().exists(tx.Ctx, clientId)
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, fmt.Errorf("looking up existence of service principal: %w", err)
	}
	if exists {
		return sp, nil
	}
	sp, err = c.servicePrincipal().register(tx)
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, fmt.Errorf("registering service principal that did not exist: %w", err)
	}
	return sp, nil
}

// Rotate rotates credentials for an existing AAD application
func (c client) Rotate(tx azure.Transaction, app azure.ApplicationResult) (*azure.ApplicationResult, error) {
	existingPasswordKeyIdsInUse := app.Password.KeyId.AllInUse
	newPasswordCredential, err := c.passwordCredential().rotate(tx, existingPasswordKeyIdsInUse)
	if err != nil {
		return nil, fmt.Errorf("rotating password credentials: %w", err)
	}
	newPasswordKeyId := string(*newPasswordCredential.KeyID)

	existingCertificateKeyIdsInUse := app.Certificate.KeyId.AllInUse
	NewCertificateKeyCredential, jwk, err := c.keyCredential().rotate(tx, existingCertificateKeyIdsInUse)
	if err != nil {
		return nil, fmt.Errorf("rotating key credentials: %w", err)
	}
	newCertificateKeyId := string(*NewCertificateKeyCredential.KeyID)

	app.Password = azure.Password{
		KeyId: azure.KeyId{
			Latest:   newPasswordKeyId,
			AllInUse: append(existingPasswordKeyIdsInUse, newPasswordKeyId),
		},
		ClientSecret: *newPasswordCredential.SecretText,
	}
	app.Certificate = azure.Certificate{
		KeyId: azure.KeyId{
			Latest:   newCertificateKeyId,
			AllInUse: append(existingCertificateKeyIdsInUse, newCertificateKeyId),
		},
		Jwk: *jwk,
	}
	return &app, nil
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
		PreAuthorizedApps:  preAuthApps,
		Tenant:             c.config.Tenant.Id,
	}, nil
}

func (c client) process(tx azure.Transaction) ([]azure.Resource, error) {
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

		/* todo: set requirement after grace period
		if err := c.servicePrincipal().setAppRoleAssignmentRequired(tx); err != nil {
			return nil, fmt.Errorf("enabling requirement for approle assignments: %w", err)
		}
		*/
	}

	return preAuthApps, nil
}
