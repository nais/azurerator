package azureadcredential

import (
	"context"
	"fmt"

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

func (r *Reconciler) ensureStatusIsValid(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	if len(credential.Status.ClientId) == 0 || len(credential.Status.ObjectId) == 0 {
		application, err := r.AzureClient.Get(ctx, *credential)
		if err != nil {
			return fmt.Errorf("failed to find object or client ID: %w", err)
		}
		credential.Status.ClientId = *application.AppID
		credential.Status.ObjectId = *application.ID
	}
	return nil
}
