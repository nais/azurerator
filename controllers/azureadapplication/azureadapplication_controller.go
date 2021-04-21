package azureadapplication

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/options"
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
	retryMaxInterval = 24 * time.Hour
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
	options        options.TransactionOptions
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
	tx, err := r.Prepare(req)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	tx.ctx = ctx
	defer cancel()

	if tx.options.Tenant.Ignore {
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

	if !tx.options.Process.Synchronize {
		return ctrl.Result{}, nil
	}

	err = r.Process(*tx)
	if err != nil {
		return r.HandleError(*tx, err)
	}

	logger.Info("successfully synchronized AzureAdApplication with Azure")
	return r.Complete(*tx)
}

func (r *Reconciler) Prepare(req ctrl.Request) (*transaction, error) {
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

	opts, err := options.NewOptions(*instance, *r.Config)
	if err != nil {
		return nil, fmt.Errorf("preparing options: %w", err)
	}

	return &transaction{
		ctx:            ctx,
		instance:       instance,
		log:            logger,
		secretDataKeys: secrets.NewSecretDataKeys(instance.Spec.SecretKeyPrefix),
		options:        opts,
	}, nil
}

func (r *Reconciler) Process(tx transaction) error {
	applicationResult, err := r.azure().createOrUpdate(tx)
	if err != nil {
		return err
	}

	r.preauthorizedapps(tx, applicationResult.PreAuthorizedApps).
		reportInvalidAsEvents()

	err = r.secrets().process(tx, applicationResult)
	if err != nil {
		return fmt.Errorf("while processing secrets: %w", err)
	}

	return nil
}

func (r *Reconciler) HandleError(tx transaction, err error) (ctrl.Result, error) {
	logger.Error(fmt.Errorf("failed to process Azure application: %w", err))
	r.reportEvent(tx, corev1.EventTypeWarning, v1.EventFailedSynchronization, "Failed to synchronize Azure application")
	metrics.IncWithNamespaceLabel(metrics.AzureAppsFailedProcessingCount, tx.instance.Namespace)

	r.reportEvent(tx, corev1.EventTypeNormal, v1.EventRetrying, "Retrying synchronization")
	return ctrl.Result{Requeue: true}, nil
}

func (r *Reconciler) Complete(tx transaction) (ctrl.Result, error) {
	metrics.IncWithNamespaceLabel(metrics.AzureAppsProcessedCount, tx.instance.Namespace)
	r.reportEvent(tx, corev1.EventTypeNormal, v1.EventSynchronized, "Azure application is up-to-date")

	annotations.RemoveAnnotation(tx.instance, annotations.ResynchronizeKey)
	annotations.RemoveAnnotation(tx.instance, annotations.RotateKey)

	tx.instance.Status.SynchronizationSecretName = tx.instance.Spec.SecretName
	now := metav1.Now()
	tx.instance.Status.SynchronizationTime = &now
	tx.instance.Status.SynchronizationTenant = r.Config.Azure.Tenant.String()

	newHash, err := tx.instance.Hash()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("calculating application hash: %w", err)
	}
	tx.instance.Status.SynchronizationHash = newHash

	err = r.updateStatus(tx)
	if err != nil {
		r.reportEvent(tx, corev1.EventTypeWarning, v1.EventFailedStatusUpdate, "Failed to update status")
		return ctrl.Result{}, err
	}

	logger.WithFields(
		log.Fields{
			"CertificateKeyIDs":  tx.instance.Status.CertificateKeyIds,
			"PasswordKeyIDs":     tx.instance.Status.PasswordKeyIds,
			"ClientID":           tx.instance.GetClientId(),
			"ObjectID":           tx.instance.GetObjectId(),
			"ServicePrincipalID": tx.instance.GetServicePrincipalId(),
		}).Info("status subresource successfully updated")

	err = r.updateAnnotations(tx)
	if err != nil {
		r.reportEvent(tx, corev1.EventTypeWarning, v1.EventRetrying, "Failed to update annotations")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
