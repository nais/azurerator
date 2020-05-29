package azureadapplication

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/secret"
)

func (r *Reconciler) create(tx transaction) (*azure.Application, error) {
	log.Info("Azure application not found, registering...")
	return r.AzureClient.Create(tx.toAzureTx())
}

func (r *Reconciler) update(tx transaction) (*azure.Application, error) {
	if err := r.ensureStatusIsValid(tx); err != nil {
		return nil, err
	}
	log.Info("Azure application already exists, updating...")
	return r.AzureClient.Update(tx.toAzureTx())
}

func (r *Reconciler) rotate(tx transaction, app azure.Application, managedSecrets secret.Lists) (*azure.Application, error) {
	var application *azure.Application
	appWithActiveKeyIds, err := secret.WithIdsFromUsedSecrets(app, managedSecrets)
	if err != nil {
		return nil, err
	}
	log.Info("rotating credentials for Azure application...")
	application, err = r.AzureClient.Rotate(tx.toAzureTx(), *appWithActiveKeyIds)
	if err != nil {
		return nil, err
	}
	application.Password.KeyId.AllInUse = append(appWithActiveKeyIds.Password.KeyId.AllInUse, application.Password.KeyId.Latest)
	application.Certificate.KeyId.AllInUse = append(appWithActiveKeyIds.Certificate.KeyId.AllInUse, application.Certificate.KeyId.Latest)
	return application, nil
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
