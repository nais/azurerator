package azureadcredential

import (
	"context"
	"fmt"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
)

// Clean up / delete all associated resources, both internal and external

func (r *Reconciler) delete(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	return r.deleteAzureApplication(ctx, credential)
}

func (r *Reconciler) deleteAzureApplication(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	log.Info("deleting Azure application...")
	exists, err := r.AzureClient.Exists(ctx, *credential)
	if err != nil {
		return err
	}
	if !exists {
		log.Info("Azure application does not exist - skipping deletion")
		return nil
	}
	if err := r.ensureStatusIsValid(ctx, credential); err != nil {
		return err
	}
	if err := r.AzureClient.Delete(ctx, *credential); err != nil {
		return fmt.Errorf("failed to delete Azure application: %w", err)
	}
	log.Info("Azure application successfully deleted")
	return nil
}
