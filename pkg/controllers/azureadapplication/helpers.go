package azureadapplication

import (
	"context"
	"fmt"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
)

func (r *Reconciler) updateStatusSubresource(ctx context.Context, resource *naisiov1alpha1.AzureAdApplication) error {
	if err := r.Status().Update(ctx, resource); err != nil {
		return fmt.Errorf("failed to update status subresource: %w", err)
	}
	return nil
}

func (r *Reconciler) ensureStatusIsValid(ctx context.Context, resource *naisiov1alpha1.AzureAdApplication) error {
	if len(resource.Status.ClientId) == 0 || len(resource.Status.ObjectId) == 0 {
		application, err := r.AzureClient.Get(ctx, *resource)
		if err != nil {
			return fmt.Errorf("failed to find object or client ID: %w", err)
		}
		resource.Status.ClientId = *application.AppID
		resource.Status.ObjectId = *application.ID
	}
	return nil
}
