package azureadapplication

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

const finalizerName string = "finalizer.azurerator.nais.io"

// Finalizers allow the controller to implement an asynchronous pre-delete hook

func (r *Reconciler) registerFinalizer(tx transaction) error {
	if !tx.instance.HasFinalizer(finalizerName) {
		log.Info("finalizer for object not found, registering...")
		tx.instance.AddFinalizer(finalizerName)
		if err := r.Update(tx.ctx, tx.instance); err != nil {
			return err
		}
		log.Info("finalizer successfully registered")
	}
	return nil
}

func (r *Reconciler) processFinalizer(tx transaction) error {
	if tx.instance.HasFinalizer(finalizerName) {
		log.Info("finalizer triggered, deleting resources...")
		if err := r.delete(tx); err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}
		r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Deleted", "Azure application is deleted")
		tx.instance.RemoveFinalizer(finalizerName)
		if err := r.Update(tx.ctx, tx.instance); err != nil {
			return fmt.Errorf("failed to remove finalizer from list: %w", err)
		}
	}
	log.Info("finalizer processed successfully")
	return nil
}
