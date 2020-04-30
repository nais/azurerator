package azureadcredential

import (
	"context"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
)

func (r *Reconciler) update(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) (azure.Application, error) {
	if err := r.ensureStatusIsValid(ctx, credential); err != nil {
		return azure.Application{}, err
	}
	if err := r.updateAzureApplication(ctx, credential); err != nil {
		return azure.Application{}, err
	}
	credential.SetStatusRotate()
	if err := r.updateStatusSubresource(ctx, credential); err != nil {
		return azure.Application{}, err
	}
	return r.rotateAzureCredentials(ctx, credential)
}

func (r *Reconciler) updateAzureApplication(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	log.Info("Azure application already exists, updating...")
	return r.AzureClient.Update(ctx, *credential)
}

func (r *Reconciler) rotateAzureCredentials(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) (azure.Application, error) {
	log.Info("Rotating credentials for Azure application...")
	return r.AzureClient.Rotate(ctx, *credential)
}
