package azureadcredential

import (
	"context"
	"fmt"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/util"
)

const finalizer string = "finalizer.azurerator.nais.io"

// Finalizers allow the controller to implement an asynchronous pre-delete hook

func (r *Reconciler) registerFinalizer(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	if !util.ContainsString(credential.ObjectMeta.Finalizers, finalizer) {
		log.Info("finalizer for object not found, registering...")
		credential.ObjectMeta.Finalizers = append(credential.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(ctx, credential); err != nil {
			return err
		}
		log.Info("finalizer successfully registered")
	}
	return nil
}

func (r *Reconciler) processFinalizer(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	if util.ContainsString(credential.ObjectMeta.Finalizers, finalizer) {
		log.Info("finalizer triggered, deleting resources...")
		// our finalizer is present, so lets handle any external dependency
		if err := r.delete(ctx, credential); err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}

		// remove our finalizer from the list and update it.
		credential.ObjectMeta.Finalizers = util.RemoveString(credential.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(ctx, credential); err != nil {
			return fmt.Errorf("failed to remove finalizer from list: %w", err)
		}
	}
	log.Info("finalizer finished successfully")
	return nil
}
