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
