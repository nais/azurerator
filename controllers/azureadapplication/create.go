package azureadapplication

import (
	"context"

	naisiov1alpha1 "github.com/nais/azureator/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
)

func (r *Reconciler) create(ctx context.Context, resource *naisiov1alpha1.AzureAdApplication) (azure.Application, error) {
	return r.createAzureApplication(ctx, resource)
}

func (r *Reconciler) createAzureApplication(ctx context.Context, resource *naisiov1alpha1.AzureAdApplication) (azure.Application, error) {
	log.Info("Azure application not found, registering...")
	resource.SetStatusNew()
	if err := r.updateStatusSubresource(ctx, resource); err != nil {
		return azure.Application{}, err
	}
	return r.AzureClient.Create(ctx, *resource)
}
