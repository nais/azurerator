package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
)

// Delete deletes the specified AAD application.
func (c client) Delete(ctx context.Context, credential v1alpha1.AzureAdCredential) error {
	exists, err := c.applicationExists(ctx, credential)
	if err != nil {
		return err
	}
	if exists {
		return c.deleteApplication(ctx, credential)
	}
	return fmt.Errorf("application does not exist: %s (clientId: %s, objectId: %s)", credential.GetUniqueName(), credential.Status.ClientId, credential.Status.ObjectId)
}

func (c client) deleteApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) error {
	var objectId string
	if len(credential.Status.ObjectId) == 0 {
		application, err := c.getApplication(ctx, credential)
		if err != nil {
			return err
		}
		objectId = *application.ID
	} else {
		objectId = credential.Status.ObjectId
	}

	if err := c.graphClient.Applications().ID(objectId).Request().Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	c.applicationsCache.Delete(credential.GetUniqueName())
	return nil
}
