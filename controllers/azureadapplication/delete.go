package azureadapplication

import (
	"fmt"
)

// Clean up / delete all associated resources, both internal and external

func (r *Reconciler) delete(tx transaction) error {
	return r.deleteAzureApplication(tx)
}

func (r *Reconciler) deleteAzureApplication(tx transaction) error {
	log.Info("deleting Azure application...")
	exists, err := r.AzureClient.Exists(tx.toAzureTx())
	if err != nil {
		return err
	}
	if !exists {
		log.Info("Azure application does not exist - skipping deletion")
		return nil
	}
	if err := r.ensureStatusIsValid(tx); err != nil {
		return err
	}
	if err := r.AzureClient.Delete(tx.toAzureTx()); err != nil {
		return fmt.Errorf("failed to delete Azure application: %w", err)
	}
	log.Info("Azure application successfully deleted")
	return nil
}
