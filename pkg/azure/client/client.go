package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
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
func (c client) Create(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	applicationResponse, err := c.registerApplication(ctx, credential)
	if err != nil {
		return azure.Application{}, err
	}
	servicePrincipal, err := c.registerServicePrincipal(ctx, applicationResponse.Application)
	if err != nil {
		return azure.Application{}, err
	}
	if err := c.registerOAuth2PermissionGrants(ctx, servicePrincipal); err != nil {
		return azure.Application{}, err
	}
	passwordCredential, err := c.addPasswordCredential(ctx, *applicationResponse.Application.ID)
	if err != nil {
		return azure.Application{}, err
	}
	if err := c.setApplicationIdentifierUri(ctx, applicationResponse.Application); err != nil {
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
		ClientId:         *applicationResponse.Application.AppID,
		ObjectId:         *applicationResponse.Application.ID,
		PasswordKeyId:    string(*passwordCredential.KeyID),
		CertificateKeyId: string(*applicationResponse.KeyCredential.KeyID),
	}, nil
}

// Delete deletes the specified AAD application.
func (c client) Delete(ctx context.Context, credential v1alpha1.AzureAdCredential) error {
	exists, err := c.Exists(ctx, credential)
	if err != nil {
		return err
	}
	if exists {
		return c.deleteApplication(ctx, credential)
	}
	return fmt.Errorf("application does not exist: %s (clientId: %s, objectId: %s)", credential.GetUniqueName(), credential.Status.ClientId, credential.Status.ObjectId)
}

// Exists returns an indication of whether the application exists in AAD or not
func (c client) Exists(ctx context.Context, credential v1alpha1.AzureAdCredential) (bool, error) {
	exists, err := c.applicationExists(ctx, credential)
	if err != nil {
		return false, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	return exists, nil
}

// Get returns a Graph API Application entity, which represents an Application in AAD
func (c client) Get(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraph.Application, error) {
	if len(credential.Status.ObjectId) == 0 {
		return c.getApplicationByName(ctx, credential)
	}
	return c.getApplicationById(ctx, credential)
}

// GetByName returns a Graph API Application entity given the displayName, which represents in Application in AAD
func (c client) GetByName(ctx context.Context, name string) (msgraph.Application, error) {
	return c.getApplicationByStringName(ctx, name)
}

// Rotate rotates credentials for an existing AAD application
func (c client) Rotate(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	clientId := credential.Status.ClientId
	objectId := credential.Status.ObjectId

	passwordCredential, err := c.rotatePasswordCredential(ctx, credential)
	if err != nil {
		return azure.Application{}, err
	}
	keyCredential, jwkPair, err := c.rotateKeyCredential(ctx, credential)
	if err != nil {
		return azure.Application{}, err
	}

	return azure.Application{
		Credentials: azure.Credentials{
			Public: azure.Public{
				ClientId: clientId,
				Jwk:      jwkPair.Public,
			},
			Private: azure.Private{
				ClientId:     clientId,
				ClientSecret: *passwordCredential.SecretText,
				Jwk:          jwkPair.Private,
			},
		},
		ClientId:         clientId,
		ObjectId:         objectId,
		CertificateKeyId: string(*keyCredential.KeyID),
		PasswordKeyId:    string(*passwordCredential.KeyID),
	}, nil
}

// Update updates an existing AAD application. Should be an idempotent operation
func (c client) Update(ctx context.Context, credential v1alpha1.AzureAdCredential) error {
	objectId := credential.Status.ObjectId
	app := util.UpdateApplicationTemplate(credential)
	if err := c.updateApplication(ctx, objectId, app); err != nil {
		return err
	}
	sp, err := c.upsertServicePrincipal(ctx, credential)
	if err != nil {
		return err
	}
	if err := c.upsertOAuth2PermissionGrants(ctx, sp); err != nil {
		return err
	}
	return nil
}
