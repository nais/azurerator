package azureadapplication

import (
	"fmt"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	finalizer2 "github.com/nais/liberator/pkg/finalizer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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

func (f finalizer) register(tx transaction) (ctrl.Result, error) {
	if !finalizer2.HasFinalizer(tx.instance, FinalizerName) {
		logger.Info("finalizer for object not found, registering...")

		controllerutil.AddFinalizer(tx.instance, FinalizerName)

		if err := f.Update(tx.ctx, tx.instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("error when registering finalizer: %w", err)
		}

		f.reportEvent(tx, corev1.EventTypeNormal, v1.EventAddedFinalizer, "Object finalizer is added")
	}
	return ctrl.Result{}, nil
}

func (f finalizer) process(tx transaction) (ctrl.Result, error) {
	if finalizer2.HasFinalizer(tx.instance, FinalizerName) {
		logger.Info("finalizer triggered, deleting resources...")

		if err := f.azure().delete(tx); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete resources: %w", err)
		}

		f.reportEvent(tx, corev1.EventTypeNormal, v1.EventDeletedInAzure, "Azure application is deleted")

		controllerutil.RemoveFinalizer(tx.instance, FinalizerName)

		if err := f.Update(tx.ctx, tx.instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to remove finalizer from list: %w", err)
		}
	}
	f.reportEvent(tx, corev1.EventTypeNormal, v1.EventDeletedFinalizer, "Object finalizer is deleted")
	metrics.IncWithNamespaceLabel(metrics.AzureAppsDeletedCount, tx.instance.Namespace)
	return ctrl.Result{}, nil
}
