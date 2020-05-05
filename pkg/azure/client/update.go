package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Update updates an existing AAD application. Should be an idempotent operation
func (c client) Update(ctx context.Context, credential v1alpha1.AzureAdCredential) error {
	objectId := credential.Status.ApplicationObjectId
	app := updateApplicationTemplate(credential)
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

func (c client) updateApplication(ctx context.Context, id string, application *msgraph.Application) error {
	if err := c.graphClient.Applications().ID(id).Request().Update(ctx, application); err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}
	return nil
}

func (c client) upsertServicePrincipal(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraphbeta.ServicePrincipal, error) {
	exists, sp, err := c.servicePrincipalExists(ctx, credential)
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, err
	}
	if exists {
		return sp, nil
	}
	application := msgraph.Application{AppID: &credential.Status.ClientId}
	sp, err = c.registerServicePrincipal(ctx, application)
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, err
	}
	return sp, err
}

func (c client) upsertOAuth2PermissionGrants(ctx context.Context, sp msgraphbeta.ServicePrincipal) error {
	exists, err := c.oAuth2PermissionGrantsExist(ctx, sp)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if err := c.registerOAuth2PermissionGrants(ctx, sp); err != nil {
		return err
	}
	return nil
}

func (c client) setApplicationIdentifierUri(ctx context.Context, application msgraph.Application) error {
	identifierUri := util.IdentifierUri(*application.AppID)
	app := util.EmptyApplication().IdentifierUri(identifierUri).Build()
	if err := c.updateApplication(ctx, *application.ID, app); err != nil {
		return fmt.Errorf("failed to add application identifier URI: %w", err)
	}
	return nil
}

// TODO - update other application metadata, preauthorizedapps
func updateApplicationTemplate(credential v1alpha1.AzureAdCredential) *msgraph.Application {
	uri := util.IdentifierUri(credential.Status.ClientId)
	return util.EmptyApplication().IdentifierUri(uri).Build()
}
