package client

import (
	"fmt"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
)

// DeleteApplication deletes the specified AAD application.
func (c client) DeleteApplication(credential v1alpha1.AzureAdCredential) error {
	exists, err := c.applicationExists(credential)
	if err != nil {
		return err
	}
	if exists {
		return c.deleteApplication(credential)
	}
	return fmt.Errorf("application does not exist: %s (clientId: %s, objectId: %s)", credential.Name, credential.Status.ClientId, credential.Status.ObjectId)
}

func (c client) deleteApplication(credential v1alpha1.AzureAdCredential) error {
	if _, err := c.applicationsClient.Delete(c.ctx, credential.Status.ObjectId); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	c.applicationsCache.Delete(credential.Name)
	return nil
}
