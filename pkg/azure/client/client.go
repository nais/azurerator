package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/msauth"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"golang.org/x/oauth2"
)

type client struct {
	config          *azure.Config
	graphClient     *msgraph.GraphServiceRequestBuilder
	graphBetaClient *msgraphbeta.GraphServiceRequestBuilder
}

func New(ctx context.Context, cfg *azure.Config) (azure.Client, error) {
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
	servicePrincipal, err := c.registerServicePrincipal(tx.Ctx, applicationResponse.Application)
	if err != nil {
		return azure.Application{}, err
	}
	if err := c.registerOAuth2PermissionGrants(tx.Ctx, servicePrincipal); err != nil {
		return azure.Application{}, err
	}
	passwordCredential, err := c.addPasswordCredential(tx.Ctx, *applicationResponse.Application.ID)
	if err != nil {
		return azure.Application{}, err
	}
	if err := c.setApplicationIdentifierUri(tx.Ctx, applicationResponse.Application); err != nil {
		return azure.Application{}, err
	}
	if err := c.addAppRoleAssignments(tx, servicePrincipal); err != nil {
		return azure.Application{}, err
	}
	preAuthApps, err := c.mapPreAuthAppsWithNames(tx.Ctx, applicationResponse.Application)
	if err != nil {
		return azure.Application{}, err
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
		return c.getApplicationByName(tx)
	}
	return c.getApplicationById(tx)
}

// GetByName returns a Graph API Application entity given the displayName, which represents in Application in AAD
func (c client) GetByName(ctx context.Context, name string) (msgraph.Application, error) {
	return c.getApplicationByStringName(ctx, name)
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

	// TODO - update other metadata for application?
	uri := util.IdentifierUri(tx.Resource.Status.ClientId)
	app := util.EmptyApplication().IdentifierUri(uri).Build()
	if err := c.updateApplication(tx.Ctx, objectId, app); err != nil {
		return azure.Application{}, err
	}
	sp, err := c.upsertServicePrincipal(tx)
	if err != nil {
		return azure.Application{}, err
	}
	if err := c.addAppRoleAssignments(tx, sp); err != nil {
		return azure.Application{}, err
	}
	if err := c.deleteRevokedAppRoleAssignments(tx, sp); err != nil {
		return azure.Application{}, err
	}
	if err := c.upsertOAuth2PermissionGrants(tx.Ctx, sp); err != nil {
		return azure.Application{}, err
	}
	preAuthApps, err := c.updatePreAuthApps(tx)
	if err != nil {
		return azure.Application{}, err
	}
	return azure.Application{
		ClientId:           clientId,
		ObjectId:           objectId,
		ServicePrincipalId: *sp.ID,
		PreAuthorizedApps:  preAuthApps,
	}, nil
}
