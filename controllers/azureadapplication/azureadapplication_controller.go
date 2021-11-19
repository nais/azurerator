package azureadapplication

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/customresources"
	"github.com/nais/azureator/pkg/kafka"
	"github.com/nais/azureator/pkg/metrics"
	"github.com/nais/azureator/pkg/reconciler"
	azureReconciler "github.com/nais/azureator/pkg/reconciler/azure"
	"github.com/nais/azureator/pkg/reconciler/finalizer"
	"github.com/nais/azureator/pkg/reconciler/namespace"
	"github.com/nais/azureator/pkg/reconciler/secrets"
	"github.com/nais/azureator/pkg/transaction"
	"github.com/nais/azureator/pkg/transaction/options"
)

const (
	retryMinInterval = 1 * time.Second
	retryMaxInterval = 15 * time.Minute
)

var appsync sync.Mutex

// Reconciler reconciles a AzureAdApplication object
type Reconciler struct {
	client.Client
	Reader            client.Reader
	Scheme            *runtime.Scheme
	AzureClient       azure.Client
	Recorder          record.EventRecorder
	Config            *config.Config
	AzureOpenIDConfig config.AzureOpenIdConfig
	KafkaProducer     kafka.Producer
}

// +kubebuilder:rbac:groups=nais.io,resources=AzureAdApplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=AzureAdApplications/status,verbs=get;update;patch;create
// +kubebuilder:rbac:groups=*,resources=events,verbs=get;list;watch;create;update

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.AzureAdApplication{}).
		WithOptions(controller.Options{RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(retryMinInterval, retryMaxInterval)}).
		WithEventFilter(eventFilterPredicate()).
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, r.Config.Controller.ContextTimeout)
	defer cancel()

	tx, err := r.Prepare(ctx, req)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if tx.Options.Tenant.Ignore {
		tx.Logger.Debugf("resource is not addressed to tenant %s, ignoring...", r.Config.Azure.Tenant)

		err := r.Azure().ProcessOrphaned(*tx)
		if err != nil {
			return r.HandleError(*tx, fmt.Errorf("processing orphaned resources: %w", err))
		}

		return ctrl.Result{}, nil
	}

	tx.Logger.Debugf("resource is addressed to tenant '%s', processing...", r.Config.Azure.Tenant.Name)

	finalizerProcessed, err := r.Finalizer().Process(*tx)
	if err != nil {
		return r.HandleError(*tx, err)
	}
	if finalizerProcessed {
		return ctrl.Result{}, nil
	}

	inSharedNamespace, err := r.Namespace().Process(tx)
	if err != nil {
		return r.HandleError(*tx, err)
	}
	if inSharedNamespace {
		metrics.IncWithNamespaceLabel(metrics.AzureAppsSkippedCount, tx.Instance.GetNamespace())
		return ctrl.Result{}, nil
	}

	// ensure that existing credentials set are in sync with Azure
	validCredentials, err := r.Azure().ValidateCredentials(*tx)
	if err != nil {
		return r.HandleError(*tx, err)
	}
	if !validCredentials {
		tx.Options.Process.Synchronize = true
		tx.Options.Process.Secret.Valid = false
	}

	err = r.Secrets().DeleteUnused(*tx)
	if err != nil {
		return r.HandleError(*tx, err)
	}

	if tx.Options.Process.Secret.Cleanup && tx.Options.Process.Secret.Valid {
		err = r.Azure().DeleteUnusedCredentials(*tx)
		if err != nil {
			return r.HandleError(*tx, err)
		}
	}

	// return early if no other operations needed
	if !tx.Options.Process.Synchronize {
		return ctrl.Result{}, nil
	}

	err = r.Process(*tx)
	if err != nil {
		return r.HandleError(*tx, err)
	}

	tx.Logger.Debug("successfully synchronized AzureAdApplication with Azure")
	return r.Complete(*tx)
}

func (r *Reconciler) Prepare(ctx context.Context, req ctrl.Request) (*transaction.Transaction, error) {
	instance := &v1.AzureAdApplication{}
	if err := r.Reader.Get(ctx, req.NamespacedName, instance); err != nil {
		return nil, err
	}

	instance.SetClusterName(r.Config.ClusterName)

	correlationId := r.getOrGenerateCorrelationId(instance)

	logger := *log.WithFields(log.Fields{
		"AzureAdApplication": req.NamespacedName,
		"CorrelationID":      correlationId,
	})

	instance.Status.CorrelationId = correlationId

	transactionSecrets, err := r.Secrets().Prepare(ctx, instance)
	if err != nil {
		return nil, fmt.Errorf("preparing transaction secrets: %w", err)
	}

	opts, err := options.NewOptions(*instance, *r.Config, *transactionSecrets)
	if err != nil {
		return nil, fmt.Errorf("preparing transaction options: %w", err)
	}

	return &transaction.Transaction{
		Ctx:      ctx,
		Instance: instance,
		Logger:   logger,
		Secrets:  *transactionSecrets,
		Options:  opts,
		ID:       correlationId,
	}, nil
}

func (r *Reconciler) Process(tx transaction.Transaction) error {
	applicationResult, err := r.Azure().Process(tx)
	if err != nil {
		return err
	}

	err = r.Secrets().Process(tx, applicationResult)
	if err != nil {
		return fmt.Errorf("while processing secrets: %w", err)
	}

	return nil
}

func (r *Reconciler) HandleError(tx transaction.Transaction, err error) (ctrl.Result, error) {
	tx.Logger.Error(fmt.Errorf("failed to process AzureAdApplication: %w", err))
	r.ReportEvent(tx, corev1.EventTypeWarning, v1.EventFailedSynchronization, "Failed to synchronize AzureAdApplication")
	metrics.IncWithNamespaceLabel(metrics.AzureAppsFailedProcessingCount, tx.Instance.Namespace)

	r.ReportEvent(tx, corev1.EventTypeNormal, v1.EventRetrying, "Retrying synchronization")
	return ctrl.Result{Requeue: true}, nil
}

func (r *Reconciler) Complete(tx transaction.Transaction) (ctrl.Result, error) {
	metrics.IncWithNamespaceLabel(metrics.AzureAppsProcessedCount, tx.Instance.Namespace)
	r.ReportEvent(tx, corev1.EventTypeNormal, v1.EventSynchronized, "Azure application is up-to-date")

	tx.Instance.Status.SynchronizationSecretName = tx.Instance.Spec.SecretName
	now := metav1.Now()
	tx.Instance.Status.SynchronizationTime = &now
	tx.Instance.Status.SynchronizationTenant = r.Config.Azure.Tenant.Id
	tx.Instance.Status.SynchronizationTenantName = r.Config.Azure.Tenant.Name

	newHash, err := tx.Instance.Hash()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("calculating application hash: %w", err)
	}
	tx.Instance.Status.SynchronizationHash = newHash

	err = r.updateStatus(tx)
	if err != nil {
		r.ReportEvent(tx, corev1.EventTypeWarning, v1.EventFailedStatusUpdate, "Failed to update status")
		return ctrl.Result{}, err
	}

	tx.Logger.WithFields(
		log.Fields{
			"CertificateKeyIDs":  tx.Instance.Status.CertificateKeyIds,
			"PasswordKeyIDs":     tx.Instance.Status.PasswordKeyIds,
			"ClientID":           tx.Instance.GetClientId(),
			"ObjectID":           tx.Instance.GetObjectId(),
			"ServicePrincipalID": tx.Instance.GetServicePrincipalId(),
		}).Debugf("status subresource successfully updated")

	err = r.updateAnnotations(tx)
	if err != nil {
		r.ReportEvent(tx, corev1.EventTypeWarning, v1.EventRetrying, "Failed to update annotations")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *Reconciler) UpdateApplication(ctx context.Context, app *v1.AzureAdApplication, updateFunc func(existing *v1.AzureAdApplication) error) error {
	appsync.Lock()
	defer appsync.Unlock()

	existing := &v1.AzureAdApplication{}
	err := r.Reader.Get(ctx, client.ObjectKey{Namespace: app.Namespace, Name: app.Name}, existing)
	if err != nil {
		return fmt.Errorf("get newest version of AzureAdApplication: %s", err)
	}

	return updateFunc(existing)
}

func (r *Reconciler) ReportEvent(tx transaction.Transaction, eventType, event, message string) {
	tx.Instance.Status.SynchronizationState = event
	r.Recorder.Event(tx.Instance, eventType, event, message)
}

func (r Reconciler) Azure() reconciler.Azure {
	return azureReconciler.NewAzureReconciler(&r, r.AzureClient, *r.Config, r.Recorder, r.KafkaProducer)
}

func (r Reconciler) Finalizer() reconciler.Finalizer {
	return finalizer.NewFinalizer(&r, r.Client)
}

func (r Reconciler) Namespace() reconciler.Namespace {
	return namespace.NewNamespaceReconciler(&r, r.Client)
}

func (r Reconciler) Secrets() reconciler.Secrets {
	return secrets.NewSecretsReconciler(&r, r.AzureOpenIDConfig, r.Client, r.Reader, r.Scheme)
}

func (r *Reconciler) updateAnnotations(tx transaction.Transaction) error {
	err := r.UpdateApplication(tx.Ctx, tx.Instance, func(existing *v1.AzureAdApplication) error {
		// remove annotations if we've already processed them.
		if customresources.HasResynchronizeAnnotation(tx.Instance) {
			annotations.RemoveAnnotation(tx.Instance, annotations.ResynchronizeKey)
			annotations.RemoveAnnotation(existing, annotations.ResynchronizeKey)
		}
		if customresources.HasRotateAnnotation(tx.Instance) {
			annotations.RemoveAnnotation(tx.Instance, annotations.RotateKey)
			annotations.RemoveAnnotation(existing, annotations.RotateKey)
		}

		merged := existing.GetAnnotations()
		for k, v := range tx.Instance.GetAnnotations() {
			merged[k] = v
		}

		existing.SetAnnotations(merged)
		return r.Update(tx.Ctx, existing)
	})
	if err != nil {
		return fmt.Errorf("updating annotations: %w", err)
	}

	return nil
}

func (r *Reconciler) updateStatus(tx transaction.Transaction) error {
	err := r.UpdateApplication(tx.Ctx, tx.Instance, func(existing *v1.AzureAdApplication) error {
		existing.Status = tx.Instance.Status
		return r.Status().Update(tx.Ctx, existing)
	})
	if err != nil {
		return fmt.Errorf("updating status fields: %w", err)
	}

	return nil
}

func (r *Reconciler) getOrGenerateCorrelationId(instance *v1.AzureAdApplication) string {
	value, found := annotations.HasAnnotation(instance, v1.DeploymentCorrelationIDAnnotation)
	if !found {
		return uuid.New().String()
	}
	return value
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
