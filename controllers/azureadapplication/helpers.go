package azureadapplication

import (
	"fmt"
)

func (r *Reconciler) updateStatusSubresource(tx transaction) error {
	if err := r.Status().Update(tx.ctx, tx.resource); err != nil {
		return fmt.Errorf("failed to update status subresource: %w", err)
	}
	return nil
}

func (r *Reconciler) ensureStatusIsValid(tx transaction) error {
	if len(tx.resource.Status.ClientId) == 0 || len(tx.resource.Status.ObjectId) == 0 {
		application, err := r.AzureClient.Get(tx.toAzureTx())
		if err != nil {
			return fmt.Errorf("failed to find object or client ID: %w", err)
		}
		tx.resource.Status.ClientId = *application.AppID
		tx.resource.Status.ObjectId = *application.ID
	}
	return nil
}
