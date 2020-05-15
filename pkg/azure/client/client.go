package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	config2 "github.com/nais/azureator/pkg/azure/config"
	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/msauth"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"golang.org/x/oauth2"
)

type client struct {
	config          *config2.Config
	graphClient     *msgraph.GraphServiceRequestBuilder
	graphBetaClient *msgraphbeta.GraphServiceRequestBuilder
}

func New(ctx context.Context, cfg *config2.Config) (azure.Client, error) {
	m := msauth.NewManager()
	scopes := []string{msauth.DefaultMSGraphScope}
	ts, err := m.ClientCredentialsGrant(ctx, cfg.Tenant, cfg.Auth.ClientId, cfg.Auth.ClientSecret, scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate graph client: %w", err)
	}

	httpClient := oauth2.NewClient(ctx, ts)
	graphClient := msgraph.NewClient(httpClient)
	graphBetaClient := msgraphbeta.NewClient(httpClient)

	return client{
		config:          cfg,
		graphClient:     graphClient,
		graphBetaClient: graphBetaClient,
	}, nil
}

// Create registers a new AAD application with all the required accompanying resources
func (c client) Create(tx azure.Transaction) (azure.Application, error) {
	applicationResponse, err := c.registerApplication(tx)
	if err != nil {
		return azure.Application{}, err
	}
	servicePrincipal, err := c.registerServicePrincipal(tx.Ctx, *applicationResponse.Application.AppID)
	if err != nil {
		return azure.Application{}, err
	}
	if err := c.registerOAuth2PermissionGrants(tx.Ctx, *servicePrincipal.ID); err != nil {
		return azure.Application{}, err
	}
	passwordCredential, err := c.addPasswordCredential(tx.Ctx, *applicationResponse.Application.ID)
	if err != nil {
		return azure.Application{}, err
	}
	if err := c.setApplicationIdentifierUri(tx.Ctx, applicationResponse.Application); err != nil {
		return azure.Application{}, err
	}
	preAuthApps, err := c.mapPreAuthAppsWithNames(tx.Ctx, applicationResponse.Application.API.PreAuthorizedApplications)
	if err != nil {
		return azure.Application{}, err
	}
	_, err = c.addAppRoleAssignments(tx, *servicePrincipal.ID, preAuthApps)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to add app role assignments: %w", err)
	}
	return azure.Application{
		Credentials: azure.Credentials{
			Public: azure.Public{
				ClientId: *applicationResponse.Application.AppID,
				Jwk:      applicationResponse.JwkPair.Public,
			},
			Private: azure.Private{
				ClientId:     *applicationResponse.Application.AppID,
				ClientSecret: *passwordCredential.SecretText,
				Jwk:          applicationResponse.JwkPair.Private,
			},
		},
		ClientId:           *applicationResponse.Application.AppID,
		ObjectId:           *applicationResponse.Application.ID,
		ServicePrincipalId: *servicePrincipal.ID,
		PasswordKeyId:      string(*passwordCredential.KeyID),
		CertificateKeyId:   string(*applicationResponse.KeyCredential.KeyID),
		PreAuthorizedApps:  preAuthApps,
	}, nil
}

// Delete deletes the specified AAD application.
func (c client) Delete(tx azure.Transaction) error {
	exists, err := c.Exists(tx)
	if err != nil {
		return err
	}
	if exists {
		return c.deleteApplication(tx)
	}
	return fmt.Errorf("application does not exist: %s (clientId: %s, objectId: %s)", tx.Resource.GetUniqueName(), tx.Resource.Status.ClientId, tx.Resource.Status.ObjectId)
}

// Exists returns an indication of whether the application exists in AAD or not
func (c client) Exists(tx azure.Transaction) (bool, error) {
	exists, err := c.applicationExists(tx)
	if err != nil {
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	return exists, nil
}

// Get returns a Graph API Application entity, which represents an Application in AAD
func (c client) Get(tx azure.Transaction) (msgraph.Application, error) {
	if len(tx.Resource.Status.ObjectId) == 0 {
		return c.GetByName(tx.Ctx, tx.Resource.GetUniqueName())
	}
	return c.getApplicationById(tx)
}

// GetByName returns a Graph API Application entity given the displayName, which represents in Application in AAD
func (c client) GetByName(ctx context.Context, name azure.DisplayName) (msgraph.Application, error) {
	return c.getApplicationByName(ctx, name)
}

// GetServicePrincipal returns the application's associated Graph ServicePrincipal entity, or registers and returns one if none exist for the application.
func (c client) GetServicePrincipal(tx azure.Transaction) (msgraphbeta.ServicePrincipal, error) {
	clientId := tx.Resource.Status.ClientId
	exists, sp, err := c.servicePrincipalExists(tx.Ctx, clientId)
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, err
	}
	if exists {
		return sp, nil
	}
	sp, err = c.registerServicePrincipal(tx.Ctx, clientId)
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, err
	}
	return sp, nil
}

// Rotate rotates credentials for an existing AAD application
func (c client) Rotate(tx azure.Transaction, app azure.Application) (azure.Application, error) {
	clientId := tx.Resource.Status.ClientId

	passwordCredential, err := c.rotatePasswordCredential(tx)
	if err != nil {
		return azure.Application{}, err
	}
	keyCredential, jwkPair, err := c.rotateKeyCredential(tx)
	if err != nil {
		return azure.Application{}, err
	}

	app.Credentials = azure.Credentials{
		Public: azure.Public{
			ClientId: clientId,
			Jwk:      jwkPair.Public,
		},
		Private: azure.Private{
			ClientId:     clientId,
			ClientSecret: *passwordCredential.SecretText,
			Jwk:          jwkPair.Private,
		},
	}
	app.CertificateKeyId = string(*keyCredential.KeyID)
	app.PasswordKeyId = string(*passwordCredential.KeyID)
	return app, nil
}

// Update updates an existing AAD application. Should be an idempotent operation
func (c client) Update(tx azure.Transaction) (azure.Application, error) {
	clientId := tx.Resource.Status.ClientId
	objectId := tx.Resource.Status.ObjectId
	spId := tx.Resource.Status.ServicePrincipalId

	// TODO - update other metadata for application?
	uri := util.IdentifierUri(tx.Resource.Status.ClientId)
	app := util.EmptyApplication().IdentifierUri(uri).Build()
	if err := c.updateApplication(tx.Ctx, objectId, app); err != nil {
		return azure.Application{}, err
	}
	if err := c.upsertOAuth2PermissionGrants(tx); err != nil {
		return azure.Application{}, err
	}
	preAuthApps, err := c.updatePreAuthApps(tx)
	if err != nil {
		return azure.Application{}, err
	}
	if err := c.updateAppRoles(tx, spId, preAuthApps); err != nil {
		return azure.Application{}, fmt.Errorf("failed to update app roles: %w", err)
	}
	return azure.Application{
		ClientId:           clientId,
		ObjectId:           objectId,
		ServicePrincipalId: spId,
		PreAuthorizedApps:  preAuthApps,
	}, nil
}
