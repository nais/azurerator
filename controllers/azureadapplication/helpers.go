package azureadapplication

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/customresources"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sync"
)

func (r *Reconciler) needsSynchronization(tx *transaction) (bool, error) {
	hashChanged, err := customresources.IsHashChanged(tx.instance)
	if err != nil {
		return false, err
	}

	secretNameChanged := customresources.SecretNameChanged(tx.instance)
	hasExpiredSecrets := customresources.HasExpiredSecrets(tx.instance, r.Config.SecretRotation.MaxAge)
	isRetrying := tx.instance.Status.SynchronizationState == v1.EventRetrying

	return hashChanged || secretNameChanged || hasExpiredSecrets || isRetrying, nil
}

func (r *Reconciler) isNotAddressedToTenant(tx *transaction) bool {
	config := r.Config.Azure.Tenant.Name
	tenant := tx.instance.Spec.Tenant

	if len(tenant) > 0 {
		return tenant != config
	}

	tenantRequired := r.Config.Validations.Tenant.Required

	if tenantRequired {
		logger.Debugf("required tenant not found in spec, skipping...")
	}

	return tenantRequired
}

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

func eventFilterPredicate() predicate.Funcs {
	return predicate.Funcs{UpdateFunc: func(event event.UpdateEvent) bool {
		objectOld := event.ObjectOld.(*v1.AzureAdApplication)
		objectNew := event.ObjectNew.(*v1.AzureAdApplication)

		specChanged := !reflect.DeepEqual(objectOld.Spec, objectNew.Spec)
		annotationsChanged := !reflect.DeepEqual(objectOld.GetAnnotations(), objectNew.GetAnnotations())
		labelsChanged := !reflect.DeepEqual(objectOld.GetLabels(), objectNew.GetLabels())
		finalizersChanged := !reflect.DeepEqual(objectOld.GetFinalizers(), objectNew.GetFinalizers())
		deletionTimestampChanged := !objectOld.GetDeletionTimestamp().Equal(objectNew.GetDeletionTimestamp())
		hashChanged := objectOld.Status.SynchronizationHash != objectNew.Status.SynchronizationHash

		return specChanged || annotationsChanged || labelsChanged || finalizersChanged || deletionTimestampChanged || hashChanged
	}}
}
