package azureadapplication

import (
	"github.com/nais/azureator/pkg/azure"
)

func (r *Reconciler) update(tx transaction) (azure.Application, error) {
	if err := r.ensureStatusIsValid(tx); err != nil {
		return azure.Application{}, err
	}
	if err := r.updateAzureApplication(tx); err != nil {
		return azure.Application{}, err
	}
	tx.resource.SetStatusRotate()
	if err := r.updateStatusSubresource(tx); err != nil {
		return azure.Application{}, err
	}
	return r.rotateAzureCredentials(tx)
}

func (r *Reconciler) updateAzureApplication(tx transaction) error {
	log.Info("Azure application already exists, updating...")
	return r.AzureClient.Update(tx.toAzureTx())
}

func (r *Reconciler) rotateAzureCredentials(tx transaction) (azure.Application, error) {
	log.Info("rotating credentials for Azure application...")
	return r.AzureClient.Rotate(tx.toAzureTx())
}
