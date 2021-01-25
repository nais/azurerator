package azureadapplication

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
)

type azureReconciler struct {
	*Reconciler
}

func (r *Reconciler) azure() azureReconciler {
	return azureReconciler{r}
}

func (a azureReconciler) create(tx transaction) (*azure.ApplicationResult, error) {
	logger.Info("Azure application not found, registering...")
	return a.AzureClient.Create(tx.toAzureTx())
}

func (a azureReconciler) update(tx transaction) (*azure.ApplicationResult, error) {
	logger.Info("Azure application already exists, updating...")
	return a.AzureClient.Update(tx.toAzureTx())
}

func (a azureReconciler) rotate(tx transaction, app azure.ApplicationResult) (*azure.ApplicationResult, error) {
	logger.Info("rotating credentials for Azure application...")
	application, err := a.AzureClient.Rotate(tx.toAzureTx(), app)
	if err != nil {
		return nil, err
	}

	logger.Info("successfully rotated credentials for Azure application")
	return application, nil
}

func (a azureReconciler) delete(tx transaction) error {
	logger.Info("deleting Azure application...")
	exists, err := a.exists(tx)
	if err != nil {
		return err
	}
	if !exists {
		logger.Info("Azure application does not exist - skipping deletion")
		return nil
	}
	if err := a.AzureClient.Delete(tx.toAzureTx()); err != nil {
		return fmt.Errorf("failed to delete Azure application: %w", err)
	}
	logger.Info("Azure application successfully deleted")
	return nil
}

func (a azureReconciler) exists(tx transaction) (bool, error) {
	exists, err := a.AzureClient.Exists(tx.toAzureTx())
	if err != nil {
		return false, fmt.Errorf("looking up existence of azure application: %w", err)
	}

	if exists {
		application, err := a.AzureClient.Get(tx.toAzureTx())
		if err != nil {
			return false, fmt.Errorf("getting azure application: %w", err)
		}
		tx.instance.SetClientId(*application.AppID)
		tx.instance.SetObjectId(*application.ID)

		sp, err := a.AzureClient.GetServicePrincipal(tx.toAzureTx())
		if err != nil {
			return false, fmt.Errorf("getting service principal for application: %w", err)
		}
		tx.instance.SetServicePrincipalId(*sp.ID)
	}

	return exists, nil
}
