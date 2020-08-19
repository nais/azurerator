package azureadapplication

import (
	"fmt"

	"github.com/nais/azureator/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
)

const FinalizerName string = "finalizer.azurerator.nais.io"

// Finalizers allow the controller to implement an asynchronous pre-delete hook

type finalizer struct {
	*Reconciler
}

func (r *Reconciler) finalizer() finalizer {
	return finalizer{r}
}

func (f finalizer) register(tx transaction) error {
	if !tx.instance.HasFinalizer(FinalizerName) {
		logger.Info("finalizer for object not found, registering...")
		tx.instance.AddFinalizer(FinalizerName)
		if err := f.Update(tx.ctx, tx.instance); err != nil {
			return err
		}
		f.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Added", "Object finalizer is added")
	}
	return nil
}

func (f finalizer) process(tx transaction) error {
	if tx.instance.HasFinalizer(FinalizerName) {
		logger.Info("finalizer triggered, deleting resources...")
		if err := f.azure().delete(tx); err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}
		f.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Deleted", "Azure application is deleted")
		tx.instance.RemoveFinalizer(FinalizerName)
		if err := f.Update(tx.ctx, tx.instance); err != nil {
			return fmt.Errorf("failed to remove finalizer from list: %w", err)
		}
	}
	f.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Deleted", "Object finalizer is deleted")
	metrics.IncWithNamespaceLabel(metrics.AzureAppsDeletedCount, tx.instance.Namespace)
	return nil
}
