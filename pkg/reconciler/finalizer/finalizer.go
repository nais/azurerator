package finalizer

import (
	"fmt"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/nais/azureator/pkg/metrics"
	"github.com/nais/azureator/pkg/reconciler"
	"github.com/nais/azureator/pkg/transaction"
	"github.com/nais/azureator/pkg/transaction/options"
)

// Finalizers allow the controller to implement an asynchronous pre-delete hook
type finalizer struct {
	reconciler.AzureAdApplication
	client client.Client
}

func NewFinalizer(reconciler reconciler.AzureAdApplication, client client.Client) reconciler.Finalizer {
	return finalizer{
		AzureAdApplication: reconciler,
		client:             client,
	}
}

func (f finalizer) Process(tx transaction.Transaction) (processed bool, err error) {
	processed = false

	if tx.Options.Finalizer.Finalize {
		err = f.finalize(tx)
		processed = true
		return
	}

	if tx.Options.Finalizer.Register {
		err = f.register(tx)
		processed = true
		return
	}

	return
}

func (f finalizer) register(tx transaction.Transaction) error {
	tx.Logger.Debug("finalizer for object not found, registering...")

	err := f.UpdateApplication(tx.Ctx, tx.Instance, func(existing *v1.AzureAdApplication) error {
		controllerutil.AddFinalizer(existing, options.FinalizerName)
		return f.client.Update(tx.Ctx, existing)
	})

	if err != nil {
		return fmt.Errorf("error when registering finalizer: %w", err)
	}

	f.ReportEvent(tx, corev1.EventTypeNormal, v1.EventAddedFinalizer, "Object finalizer is added")
	return nil
}

func (f finalizer) finalize(tx transaction.Transaction) error {
	if tx.Options.Finalizer.Register {
		return nil
	}

	tx.Logger.Debug("finalizer triggered, deleting resources...")

	if tx.Options.Finalizer.DeleteFromAzure {
		err := f.Azure().Delete(tx)
		if err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}

		f.ReportEvent(tx, corev1.EventTypeNormal, v1.EventDeletedInAzure, "Azure application is deleted")
	} else {
		err := f.Azure().PurgeCredentials(tx)
		if err != nil {
			return fmt.Errorf("purging credentials from Azure AD: %w", err)
		}
	}

	err := f.UpdateApplication(tx.Ctx, tx.Instance, func(existing *v1.AzureAdApplication) error {
		controllerutil.RemoveFinalizer(existing, options.FinalizerName)
		return f.client.Update(tx.Ctx, existing)
	})
	if err != nil {
		return fmt.Errorf("failed to remove finalizer from list: %w", err)
	}

	f.ReportEvent(tx, corev1.EventTypeNormal, v1.EventDeletedFinalizer, "Object finalizer is deleted")
	metrics.IncWithNamespaceLabel(metrics.AzureAppsDeletedCount, tx.Instance.Namespace)

	return nil
}
