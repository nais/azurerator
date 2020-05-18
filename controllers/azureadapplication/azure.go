package azureadapplication

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
)

func (r *Reconciler) create(tx transaction) (azure.Application, error) {
	log.Info("Azure application not found, registering...")
	tx.instance.SetStatusNew()
	if err := r.updateStatusSubresource(tx); err != nil {
		return azure.Application{}, err
	}
	return r.AzureClient.Create(tx.toAzureTx())
}

func (r *Reconciler) update(tx transaction) (azure.Application, error) {
	if err := r.ensureStatusIsValid(tx); err != nil {
		return azure.Application{}, err
	}
	log.Info("Azure application already exists, updating...")
	app, err := r.AzureClient.Update(tx.toAzureTx())
	if err != nil {
		return azure.Application{}, err
	}

	// todo - separate rotate operation?
	tx.instance.SetStatusRotate()
	if err := r.updateStatusSubresource(tx); err != nil {
		return azure.Application{}, err
	}
	log.Info("rotating credentials for Azure application...")
	return r.AzureClient.Rotate(tx.toAzureTx(), app)
}

func (r *Reconciler) delete(tx transaction) error {
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
