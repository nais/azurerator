package azureadapplication

import (
	"fmt"
	finalizer2 "github.com/nais/azureator/pkg/finalizers"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/nais/azureator/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
)

// Finalizers allow the controller to implement an asynchronous pre-delete hook

type finalizer struct {
	*Reconciler
}

func (r *Reconciler) finalizer() finalizer {
	return finalizer{r}
}

func (f finalizer) process(tx transaction) (processed bool, err error) {
	processed = false

	if tx.options.Finalizer.Finalize {
		err = f.finalize(tx)
		processed = true
		return
	}

	if tx.options.Finalizer.Register {
		err = f.register(tx)
		processed = true
		return
	}

	return
}

func (f finalizer) register(tx transaction) error {
	logger.Debug("finalizer for object not found, registering...")

	err := f.updateApplication(tx.ctx, tx.instance, func(existing *v1.AzureAdApplication) error {
		controllerutil.AddFinalizer(existing, finalizer2.Name)
		return f.Update(tx.ctx, existing)
	})

	if err != nil {
		return fmt.Errorf("error when registering finalizer: %w", err)
	}

	f.reportEvent(tx, corev1.EventTypeNormal, v1.EventAddedFinalizer, "Object finalizer is added")
	return nil
}

func (f finalizer) finalize(tx transaction) error {
	if tx.options.Finalizer.Register {
		return nil
	}

	logger.Debug("finalizer triggered, deleting resources...")

	if tx.options.Finalizer.DeleteFromAzure {
		err := f.azure().delete(tx)
		if err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}

		f.reportEvent(tx, corev1.EventTypeNormal, v1.EventDeletedInAzure, "Azure application is deleted")
	}

	err := f.updateApplication(tx.ctx, tx.instance, func(existing *v1.AzureAdApplication) error {
		controllerutil.RemoveFinalizer(existing, finalizer2.Name)
		return f.Update(tx.ctx, existing)
	})
	if err != nil {
		return fmt.Errorf("failed to remove finalizer from list: %w", err)
	}

	f.reportEvent(tx, corev1.EventTypeNormal, v1.EventDeletedFinalizer, "Object finalizer is deleted")
	metrics.IncWithNamespaceLabel(metrics.AzureAppsDeletedCount, tx.instance.Namespace)

	return nil
}
