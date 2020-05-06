package azureadapplication

import (
	"context"
	"fmt"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
)

// Clean up / delete all associated resources, both internal and external

func (r *Reconciler) delete(ctx context.Context, resource *naisiov1alpha1.AzureAdApplication) error {
	return r.deleteAzureApplication(ctx, resource)
}

func (r *Reconciler) deleteAzureApplication(ctx context.Context, resource *naisiov1alpha1.AzureAdApplication) error {
	log.Info("deleting Azure application...")
	exists, err := r.AzureClient.Exists(ctx, *resource)
	if err != nil {
		return err
	}
	if !exists {
		log.Info("Azure application does not exist - skipping deletion")
		return nil
	}
	if err := r.ensureStatusIsValid(ctx, resource); err != nil {
		return err
	}
	if err := r.AzureClient.Delete(ctx, *resource); err != nil {
		return fmt.Errorf("failed to delete Azure application: %w", err)
	}
	log.Info("Azure application successfully deleted")
	return nil
}
