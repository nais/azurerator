package azureadapplication

import (
	"fmt"

	"github.com/nais/azureator/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
)

const FinalizerName string = "finalizer.azurerator.nais.io"

// Finalizers allow the controller to implement an asynchronous pre-delete hook

func (r *Reconciler) registerFinalizer(tx transaction) error {
	if !tx.instance.HasFinalizer(FinalizerName) {
		logger.Info("finalizer for object not found, registering...")
		tx.instance.AddFinalizer(FinalizerName)
		if err := r.Update(tx.ctx, tx.instance); err != nil {
			return err
		}
		r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Added", "Object finalizer is added")
	}
	return nil
}

func (r *Reconciler) processFinalizer(tx transaction) error {
	if tx.instance.HasFinalizer(FinalizerName) {
		logger.Info("finalizer triggered, deleting resources...")
		if err := r.delete(tx); err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}
		r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Deleted", "Azure application is deleted")
		tx.instance.RemoveFinalizer(FinalizerName)
		if err := r.Update(tx.ctx, tx.instance); err != nil {
			return fmt.Errorf("failed to remove finalizer from list: %w", err)
		}
	}
	r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Deleted", "Object finalizer is deleted")
	metrics.IncWithNamespaceLabel(metrics.AzureAppsDeletedCount, tx.instance.Namespace)
	return nil
}
