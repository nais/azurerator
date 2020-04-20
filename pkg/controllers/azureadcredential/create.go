package azureadcredential

import (
	"context"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
)

func (r *Reconciler) create(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) (azure.Application, error) {
	return r.createAzureApplication(ctx, credential)
}

func (r *Reconciler) createAzureApplication(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) (azure.Application, error) {
	log.Info("Azure application not found, registering...")
	credential.StatusNewProvisioning()
	if err := r.updateStatusSubresource(ctx, credential); err != nil {
		return azure.Application{}, err
	}
	return r.AzureClient.Create(ctx, *credential)
}
