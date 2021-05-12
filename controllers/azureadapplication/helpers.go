package azureadapplication

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/google/uuid"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/nais/azureator/pkg/annotations"
)

var appsync sync.Mutex

func (r *Reconciler) updateApplication(ctx context.Context, app *v1.AzureAdApplication, updateFunc func(existing *v1.AzureAdApplication) error) error {
	appsync.Lock()
	defer appsync.Unlock()

	existing := &v1.AzureAdApplication{}
	err := r.Reader.Get(ctx, client.ObjectKey{Namespace: app.Namespace, Name: app.Name}, existing)
	if err != nil {
		return fmt.Errorf("get newest version of AzureAdApplication: %s", err)
	}

	return updateFunc(existing)
}

func (r *Reconciler) updateAnnotations(tx transaction) error {
	err := r.updateApplication(tx.ctx, tx.instance, func(existing *v1.AzureAdApplication) error {
		existing.SetAnnotations(tx.instance.GetAnnotations())
		return r.Update(tx.ctx, existing)
	})
	if err != nil {
		return fmt.Errorf("updating annotations: %w", err)
	}

	return nil
}

func (r *Reconciler) updateStatus(tx transaction) error {
	err := r.updateApplication(tx.ctx, tx.instance, func(existing *v1.AzureAdApplication) error {
		existing.Status = tx.instance.Status
		return r.Status().Update(tx.ctx, existing)
	})
	if err != nil {
		return fmt.Errorf("updating status fields: %w", err)
	}

	return nil
}

func (r *Reconciler) getOrGenerateCorrelationId(instance *v1.AzureAdApplication) string {
	value, found := annotations.HasAnnotation(instance, nais_io_v1alpha1.DeploymentCorrelationIDAnnotation)
	if !found {
		return uuid.New().String()
	}
	return value
}

func (r *Reconciler) reportEvent(tx transaction, eventType, event, message string) {
	tx.instance.Status.SynchronizationState = event
	r.Recorder.Event(tx.instance, eventType, event, message)
}

func eventFilterPredicate() predicate.Funcs {
	return predicate.Funcs{UpdateFunc: func(event event.UpdateEvent) bool {
		objectOld := event.ObjectOld.(*v1.AzureAdApplication)
		objectNew := event.ObjectNew.(*v1.AzureAdApplication)

		specChanged := !reflect.DeepEqual(objectOld.Spec, objectNew.Spec)
		annotationsChanged := !reflect.DeepEqual(objectOld.GetAnnotations(), objectNew.GetAnnotations())
		labelsChanged := !reflect.DeepEqual(objectOld.GetLabels(), objectNew.GetLabels())
		finalizersChanged := !reflect.DeepEqual(objectOld.GetFinalizers(), objectNew.GetFinalizers())
		deletionTimestampChanged := !objectOld.GetDeletionTimestamp().Equal(objectNew.GetDeletionTimestamp())

		return specChanged || annotationsChanged || labelsChanged || finalizersChanged || deletionTimestampChanged
	}}
}
