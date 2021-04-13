package azureadapplication

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/config"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"time"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/metrics"
	"github.com/nais/azureator/pkg/secrets"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	contextTimeout   = 1 * time.Minute
	retryMinInterval = 15 * time.Second
	retryMaxInterval = 1 * time.Minute
)

// Reconciler reconciles a AzureAdApplication object
type Reconciler struct {
	client.Client
	Reader            client.Reader
	Scheme            *runtime.Scheme
	AzureClient       azure.Client
	Recorder          record.EventRecorder
	Config            *config.Config
	AzureOpenIDConfig config.AzureOpenIdConfig
}

type transaction struct {
	ctx            context.Context
	instance       *v1.AzureAdApplication
	log            log.Entry
	secretDataKeys secrets.SecretDataKeys
}

func (t *transaction) toAzureTx() azure.Transaction {
	return azure.Transaction{
		Ctx:      t.ctx,
		Instance: *t.instance,
		Log:      t.log,
	}
}

var logger log.Entry
var correlationId string

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

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	tx, err := r.prepare(req)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	tx.ctx = ctx
	defer cancel()

	if r.isNotAddressedToTenant(tx) {
		logger.Debugf("resource is not addressed to tenant '%s', ignoring...", r.Config.Azure.Tenant.Name)
		return ctrl.Result{}, nil
	}

	logger.Debugf("resource is addressed to tenant '%s', processing...", r.Config.Azure.Tenant.Name)

	finalizerProcessed, err := r.finalizer().process(*tx)
	if err != nil {
		return ctrl.Result{}, err
	}
	if finalizerProcessed {
		return ctrl.Result{}, nil
	}

	inSharedNamespace, err := r.namespaces().process(tx)
	if err != nil {
		return ctrl.Result{}, err
	}
	if inSharedNamespace {
		metrics.IncWithNamespaceLabel(metrics.AzureAppsSkippedCount, tx.instance.GetNamespace())
		return ctrl.Result{}, nil
	}

	needsSynchronization, err := r.needsSynchronization(tx)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !needsSynchronization {
		return ctrl.Result{}, nil
	}

	requeue, err := r.process(*tx)
	if err != nil {
		return r.handleError(*tx, err)
	}

	logger.Info("successfully synchronized AzureAdApplication with Azure")
	return r.complete(*tx, requeue)
}

func (r *Reconciler) prepare(req ctrl.Request) (*transaction, error) {
	ctx := context.Background()

	instance := &v1.AzureAdApplication{}
	if err := r.Reader.Get(ctx, req.NamespacedName, instance); err != nil {
		return nil, err
	}

	instance.SetClusterName(r.Config.ClusterName)

	correlationId = r.getOrGenerateCorrelationId(instance)

	logger = *log.WithFields(log.Fields{
		"AzureAdApplication": req.NamespacedName,
		"correlation_id":     correlationId,
	})

	instance.Status.CorrelationId = correlationId

	return &transaction{
		ctx:            ctx,
		instance:       instance,
		log:            logger,
		secretDataKeys: secrets.NewSecretDataKeys(instance.Spec.SecretKeyPrefix),
	}, nil
}

func (r *Reconciler) getOrGenerateCorrelationId(instance *v1.AzureAdApplication) string {
	value, found := annotations.HasAnnotation(instance, nais_io_v1alpha1.DeploymentCorrelationIDAnnotation)
	if !found {
		return uuid.New().String()
	}
	return value
}

func (r *Reconciler) process(tx transaction) (bool, error) {
	requeue := false

	applicationResult, err := r.azure().createOrUpdate(tx)
	if err != nil {
		return true, err
	}

	preAuthorizedApps := r.preauthorizedapps(tx.instance, applicationResult.PreAuthorizedApps)
	preAuthorizedApps.reportInvalidAsEvents()

	if preAuthorizedApps.shouldRequeueSynchronization() {
		requeue = true
	}

	err = r.secrets().process(tx, applicationResult)
	if err != nil {
		return true, fmt.Errorf("while processing secrets: %w", err)
	}

	return requeue, nil
}

func (r *Reconciler) handleError(tx transaction, err error) (ctrl.Result, error) {
	logger.Error(fmt.Errorf("failed to process Azure application: %w", err))
	r.reportEvent(tx, corev1.EventTypeWarning, v1.EventFailedSynchronization, "Failed to synchronize Azure application")
	metrics.IncWithNamespaceLabel(metrics.AzureAppsFailedProcessingCount, tx.instance.Namespace)

	r.reportEvent(tx, corev1.EventTypeNormal, v1.EventRetrying, "Retrying synchronization")
	return ctrl.Result{Requeue: true}, nil
}

func (r *Reconciler) complete(tx transaction, requeue bool) (ctrl.Result, error) {
	metrics.IncWithNamespaceLabel(metrics.AzureAppsProcessedCount, tx.instance.Namespace)
	r.reportEvent(tx, corev1.EventTypeNormal, v1.EventSynchronized, "Azure application is up-to-date")

	if requeue {
		r.reportEvent(tx, corev1.EventTypeWarning, v1.EventRetrying,
			"Azure application is up-to-date, but spec contains invalid pre-authorized apps. "+
				"Retrying synchronization with exponential backoff...",
		)
	}

	err := r.updateStatus(tx)
	if err != nil {
		r.reportEvent(tx, corev1.EventTypeWarning, v1.EventFailedStatusUpdate, "Failed to update status")
		return ctrl.Result{}, err
	}

	if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) updateStatus(tx transaction) error {
	tx.instance.Status.SynchronizationSecretName = tx.instance.Spec.SecretName
	now := metav1.Now()
	tx.instance.Status.SynchronizationTime = &now
	tx.instance.Status.SynchronizationTenant = r.Config.Azure.Tenant.String()

	newHash, err := tx.instance.Hash()
	if err != nil {
		return fmt.Errorf("calculating application hash: %w", err)
	}
	tx.instance.Status.SynchronizationHash = newHash

	err = r.updateApplication(tx.ctx, tx.instance, func(existing *v1.AzureAdApplication) error {
		existing.Status = tx.instance.Status
		return r.Status().Update(tx.ctx, existing)
	})
	if err != nil {
		return fmt.Errorf("updating status fields: %w", err)
	}

	logger.WithFields(
		log.Fields{
			"CertificateKeyIDs":  tx.instance.Status.CertificateKeyIds,
			"PasswordKeyIDs":     tx.instance.Status.PasswordKeyIds,
			"ClientID":           tx.instance.GetClientId(),
			"ObjectID":           tx.instance.GetObjectId(),
			"ServicePrincipalID": tx.instance.GetServicePrincipalId(),
		}).Info("status subresource successfully updated")

	return nil
}

func (r *Reconciler) reportEvent(tx transaction, eventType, event, message string) {
	tx.instance.Status.SynchronizationState = event
	r.Recorder.Event(tx.instance, eventType, event, message)
}
