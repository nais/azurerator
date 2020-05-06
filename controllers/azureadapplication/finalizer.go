package azureadapplication

import (
	"context"
	"fmt"

	naisiov1alpha1 "github.com/nais/azureator/apis/v1alpha1"
	"github.com/nais/azureator/pkg/util"
)

const finalizer string = "finalizer.azurerator.nais.io"

// Finalizers allow the controller to implement an asynchronous pre-delete hook

func (r *Reconciler) registerFinalizer(ctx context.Context, resource *naisiov1alpha1.AzureAdApplication) error {
	if !util.ContainsString(resource.ObjectMeta.Finalizers, finalizer) {
		log.Info("finalizer for object not found, registering...")
		resource.ObjectMeta.Finalizers = append(resource.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(ctx, resource); err != nil {
			return err
		}
		log.Info("finalizer successfully registered")
	}
	return nil
}

func (r *Reconciler) processFinalizer(ctx context.Context, resource *naisiov1alpha1.AzureAdApplication) error {
	if util.ContainsString(resource.ObjectMeta.Finalizers, finalizer) {
		log.Info("finalizer triggered, deleting resources...")
		// our finalizer is present, so lets handle any external dependency
		if err := r.delete(ctx, resource); err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}

		// remove our finalizer from the list and update it.
		resource.ObjectMeta.Finalizers = util.RemoveString(resource.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(ctx, resource); err != nil {
			return fmt.Errorf("failed to remove finalizer from list: %w", err)
		}
	}
	log.Info("finalizer finished successfully")
	return nil
}
