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
	return r.updateAzureApplication(ctx, credential)
}

func (r *Reconciler) updateAzureApplication(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) (azure.Application, error) {
	log.Info("Azure application already exists, updating...")
	credential.SetStatusRotate()
	if err := r.updateStatusSubresource(ctx, credential); err != nil {
		return azure.Application{}, err
	}
	return r.AzureClient.Update(ctx, *credential)
}
