package azureadapplication

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/secrets"
)

func (r *Reconciler) create(tx transaction) (*azure.Application, error) {
	logger.Info("Azure application not found, registering...")
	return r.AzureClient.Create(tx.toAzureTx())
}

func (r *Reconciler) update(tx transaction) (*azure.Application, error) {
	if err := r.ensureStatusIsValid(tx); err != nil {
		return nil, err
	}
	logger.Info("Azure application already exists, updating...")
	return r.AzureClient.Update(tx.toAzureTx())
}

func (r *Reconciler) rotate(tx transaction, app azure.Application, managedSecrets secrets.Lists) (*azure.Application, error) {
	appWithActiveKeyIds := secrets.WithIdsFromUsedSecrets(app, managedSecrets)
	logger.Info("rotating credentials for Azure application...")
	application, err := r.AzureClient.Rotate(tx.toAzureTx(), appWithActiveKeyIds)
	if err != nil {
		return nil, err
	}
	application.Password.KeyId.AllInUse = append(appWithActiveKeyIds.Password.KeyId.AllInUse, application.Password.KeyId.Latest)
	application.Certificate.KeyId.AllInUse = append(appWithActiveKeyIds.Certificate.KeyId.AllInUse, application.Certificate.KeyId.Latest)
	logger.Info("successfully rotated credentials for Azure application")
	return application, nil
}

func (r *Reconciler) delete(tx transaction) error {
	logger.Info("deleting Azure application...")
	exists, err := r.AzureClient.Exists(tx.toAzureTx())
	if err != nil {
		return err
	}
	if !exists {
		logger.Info("Azure application does not exist - skipping deletion")
		return nil
	}
	if err := r.ensureStatusIsValid(tx); err != nil {
		return err
	}
	if err := r.AzureClient.Delete(tx.toAzureTx()); err != nil {
		return fmt.Errorf("failed to delete Azure application: %w", err)
	}
	logger.Info("Azure application successfully deleted")
	return nil
}
