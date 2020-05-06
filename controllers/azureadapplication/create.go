package azureadapplication

import (
	"github.com/nais/azureator/pkg/azure"
)

func (r *Reconciler) create(tx transaction) (azure.Application, error) {
	return r.createAzureApplication(tx)
}

func (r *Reconciler) createAzureApplication(tx transaction) (azure.Application, error) {
	log.Info("Azure application not found, registering...")
	tx.resource.SetStatusNew()
	if err := r.updateStatusSubresource(tx); err != nil {
		return azure.Application{}, err
	}
	return r.AzureClient.Create(tx.toAzureTx())
}
