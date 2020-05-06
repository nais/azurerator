package azureadapplication

import (
	"fmt"

	"github.com/nais/azureator/pkg/util"
)

const finalizer string = "finalizer.azurerator.nais.io"

// Finalizers allow the controller to implement an asynchronous pre-delete hook

func (r *Reconciler) registerFinalizer(tx transaction) error {
	if !util.ContainsString(tx.resource.ObjectMeta.Finalizers, finalizer) {
		log.Info("finalizer for object not found, registering...")
		tx.resource.ObjectMeta.Finalizers = append(tx.resource.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(tx.ctx, tx.resource); err != nil {
			return err
		}
		log.Info("finalizer successfully registered")
	}
	return nil
}

func (r *Reconciler) processFinalizer(tx transaction) error {
	if util.ContainsString(tx.resource.ObjectMeta.Finalizers, finalizer) {
		log.Info("finalizer triggered, deleting resources...")
		// our finalizer is present, so lets handle any external dependency
		if err := r.delete(tx); err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}

		// remove our finalizer from the list and update it.
		tx.resource.ObjectMeta.Finalizers = util.RemoveString(tx.resource.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(tx.ctx, tx.resource); err != nil {
			return fmt.Errorf("failed to remove finalizer from list: %w", err)
		}
	}
	log.Info("finalizer finished successfully")
	return nil
}
