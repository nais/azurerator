package finalizer

import (
	"fmt"

	"github.com/nais/azureator/pkg/annotations"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/nais/azureator/pkg/metrics"
	"github.com/nais/azureator/pkg/reconciler"
	"github.com/nais/azureator/pkg/transaction"
)

const (
	Name string = "azure.nais.io/finalizer"
	// OldName is not domain-qualified and triggers a warning from the API server, use FinalizerName instead.
	// TODO: remove once no instances with the old finalizer exist.
	OldName string = "finalizer.azurerator.nais.io"
)

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

func (f finalizer) Process(tx transaction.Transaction) (bool, error) {
	hasFinalizer := controllerutil.ContainsFinalizer(tx.Instance, Name)
	hasOldFinalizer := controllerutil.ContainsFinalizer(tx.Instance, OldName)
	shouldFinalize := !tx.Instance.GetDeletionTimestamp().IsZero()

	if (hasFinalizer || hasOldFinalizer) && shouldFinalize {
		return true, f.finalize(tx)
	}

	if !hasFinalizer {
		return true, f.register(tx)
	}

	return false, nil
}

func (f finalizer) register(tx transaction.Transaction) error {
	tx.Logger.Debug("finalizer for object not found, registering...")

	err := f.UpdateApplication(tx.Ctx, tx.Instance, func(existing *v1.AzureAdApplication) error {
		controllerutil.AddFinalizer(existing, Name)
		return f.client.Update(tx.Ctx, existing)
	})
	if err != nil {
		return fmt.Errorf("error when registering finalizer: %w", err)
	}

	return nil
}

func (f finalizer) finalize(tx transaction.Transaction) error {
	tx.Logger.Debug("finalizer triggered, deleting resources...")

	_, shouldPreserve := annotations.HasAnnotation(tx.Instance, annotations.PreserveKey)
	if shouldPreserve {
		err := f.Azure().PurgeCredentials(tx)
		if err != nil {
			return fmt.Errorf("purging credentials from Azure AD: %w", err)
		}
	} else {
		err := f.Azure().Delete(tx)
		if err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}

		f.ReportEvent(tx, corev1.EventTypeNormal, v1.EventDeletedInAzure, "Azure application is deleted")
	}

	err := f.UpdateApplication(tx.Ctx, tx.Instance, func(existing *v1.AzureAdApplication) error {
		controllerutil.RemoveFinalizer(existing, Name)
		// TODO: remove once old finalizer is no longer in use
		controllerutil.RemoveFinalizer(existing, OldName)
		return f.client.Update(tx.Ctx, existing)
	})
	if err != nil {
		return fmt.Errorf("failed to remove finalizer from list: %w", err)
	}

	metrics.IncWithNamespaceLabel(metrics.AzureAppsDeletedCount, tx.Instance.Namespace)
	return nil
}
