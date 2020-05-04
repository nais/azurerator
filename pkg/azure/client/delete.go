package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
)

// Delete deletes the specified AAD application.
func (c client) Delete(ctx context.Context, credential v1alpha1.AzureAdCredential) error {
	exists, err := c.Exists(ctx, credential)
	if err != nil {
		return err
	}
	if exists {
		return c.deleteApplication(ctx, credential)
	}
	return fmt.Errorf("application does not exist: %s (clientId: %s, objectId: %s)", credential.GetUniqueName(), credential.Status.ClientId, credential.Status.ApplicationObjectId)
}

func (c client) deleteApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) error {
	if err := c.graphClient.Applications().ID(credential.Status.ApplicationObjectId).Request().Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	return nil
}
