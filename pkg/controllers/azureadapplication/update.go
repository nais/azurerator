package azureadapplication

import (
	"context"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
)

func (r *Reconciler) update(ctx context.Context, resource *naisiov1alpha1.AzureAdApplication) (azure.Application, error) {
	if err := r.ensureStatusIsValid(ctx, resource); err != nil {
		return azure.Application{}, err
	}
	if err := r.updateAzureApplication(ctx, resource); err != nil {
		return azure.Application{}, err
	}
	resource.SetStatusRotate()
	if err := r.updateStatusSubresource(ctx, resource); err != nil {
		return azure.Application{}, err
	}
	return r.rotateAzureCredentials(ctx, resource)
}

func (r *Reconciler) updateAzureApplication(ctx context.Context, resource *naisiov1alpha1.AzureAdApplication) error {
	log.Info("Azure application already exists, updating...")
	return r.AzureClient.Update(ctx, *resource)
}

func (r *Reconciler) rotateAzureCredentials(ctx context.Context, resource *naisiov1alpha1.AzureAdApplication) (azure.Application, error) {
	log.Info("rotating credentials for Azure application...")
	return r.AzureClient.Rotate(ctx, *resource)
}
