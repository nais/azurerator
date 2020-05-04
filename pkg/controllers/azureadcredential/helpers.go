package azureadcredential

import (
	"context"
	"fmt"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
)

func (r *Reconciler) updateStatusSubresource(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	if err := r.Status().Update(ctx, credential); err != nil {
		return fmt.Errorf("failed to update status subresource: %w", err)
	}
	return nil
}

func (r *Reconciler) ensureStatusIsValid(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	if len(credential.Status.ClientId) == 0 || len(credential.Status.ApplicationObjectId) == 0 {
		application, err := r.AzureClient.Get(ctx, *credential)
		if err != nil {
			return fmt.Errorf("failed to find object or client ID: %w", err)
		}
		credential.Status.ClientId = *application.AppID
		credential.Status.ApplicationObjectId = *application.ID
	}
	return nil
}
