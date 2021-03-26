package azureadapplication

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/customresources"
	finalizer2 "github.com/nais/liberator/pkg/finalizer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"strings"
	"time"

	"github.com/google/uuid"
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

// AzureAdApplicationReconciler reconciles a AzureAdApplication object
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
	ctx      context.Context
	instance *v1.AzureAdApplication
	log      log.Entry
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

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	correlationId = uuid.New().String()

	logger = *log.WithFields(log.Fields{
		"AzureAdApplication": req.NamespacedName,
		"correlationId":      correlationId,
	})

	tx, err := r.prepare(req)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	tx.ctx = ctx
	defer cancel()

	if r.shouldSkip(tx) {
		return ctrl.Result{}, nil
	}

	if finalizer2.IsBeingDeleted(tx.instance) {
		return r.finalizer().process(*tx)
	}

	if !finalizer2.HasFinalizer(tx.instance, FinalizerName) {
		return r.finalizer().register(*tx)
	}

	inSharedNamespace, err := r.inSharedNamespace(tx)
	if err != nil {
		return ctrl.Result{}, err
	}

	if inSharedNamespace {
		if err := r.Client.Status().Update(tx.ctx, tx.instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update resource with skip flag: %w", err)
		}

		if err := r.Client.Update(tx.ctx, tx.instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update resource with skip flag: %w", err)
		}

		metrics.IncWithNamespaceLabel(metrics.AzureAppsSkippedCount, tx.instance.Namespace)
		return ctrl.Result{}, nil
	}

	hashChanged, err := customresources.IsHashChanged(tx.instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !hashChanged && !customresources.ShouldRotateSecrets(tx.instance, r.Config.MaxSecretAge) {
		return ctrl.Result{}, nil
	}

	if err := r.process(*tx); err != nil {
		return r.handleError(*tx, err)
	}

	logger.Info("successfully synchronized AzureAdApplication with Azure")
	return r.complete(*tx)
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.AzureAdApplication{}).
		WithOptions(controller.Options{RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(retryMinInterval, retryMaxInterval)}).
		Complete(r)
}

func (r *Reconciler) prepare(req ctrl.Request) (*transaction, error) {
	ctx := context.Background()

	instance := &v1.AzureAdApplication{}
	if err := r.Reader.Get(ctx, req.NamespacedName, instance); err != nil {
		return nil, err
	}
	instance.SetClusterName(r.Config.ClusterName)
	instance.Status.CorrelationId = correlationId

	return &transaction{ctx, instance, logger}, nil
}

func (r *Reconciler) process(tx transaction) error {
	applicationResult, err := r.azure().createOrUpdate(tx)
	if err != nil {
		return err
	}

	r.preauthorizedapps(tx.instance, applicationResult.PreAuthorizedApps).
		filterInvalid().
		reportInvalidAsEvents()

	secretClient := r.secrets(&tx)

	managedSecrets, err := secretClient.GetManaged()
	if err != nil {
		return fmt.Errorf("getting managed secrets: %w", err)
	}

	keyIdsInUse := secrets.ExtractKeyIdsInUse(*managedSecrets)
	credentialsSet, validCredentials, err := secrets.ExtractCredentialsSetFromSecretLists(*managedSecrets, tx.instance.Status.SynchronizationSecretName)
	if err != nil {
		return fmt.Errorf("extracting credentials set from secret: %w", err)
	}

	shouldRotateSecrets := customresources.ShouldRotateSecrets(tx.instance, r.Config.MaxSecretAge)
	shouldUpdateSecrets := customresources.ShouldUpdateSecrets(tx.instance, r.Config.MaxSecretAge)

	validCredentials = validCredentials && strings.Contains(tx.instance.Status.SynchronizationTenant, r.Config.Azure.Tenant.Name)

	if validCredentials && !shouldUpdateSecrets {
		return nil
	}

	if !validCredentials {
		credentialsSet, err = r.azure().addCredentials(tx, &keyIdsInUse)
		if err != nil {
			return fmt.Errorf("adding azure credentials: %w", err)
		}
	} else if shouldRotateSecrets {
		credentialsSet, err = r.azure().rotateCredentials(tx, *credentialsSet, &keyIdsInUse)
		if err != nil {
			return fmt.Errorf("rotating azure credentials: %w", err)
		}
	}

	if err := secretClient.CreateOrUpdate(*applicationResult, *credentialsSet, r.AzureOpenIDConfig); err != nil {
		return err
	}

	if err := secretClient.DeleteUnused(managedSecrets.Unused); err != nil {
		return err
	}

	tx.instance.Status.CertificateKeyIds = keyIdsInUse.Certificate
	tx.instance.Status.PasswordKeyIds = keyIdsInUse.Password

	if !validCredentials || shouldRotateSecrets {
		now := metav1.Now()
		tx.instance.Status.SynchronizationSecretRotationTime = &now
	}

	return nil
}

func (r *Reconciler) handleError(tx transaction, err error) (ctrl.Result, error) {
	logger.Error(fmt.Errorf("failed to process Azure application: %w", err))
	r.reportEvent(tx, corev1.EventTypeWarning, v1.EventFailedSynchronization, "Failed to synchronize Azure application")
	metrics.IncWithNamespaceLabel(metrics.AzureAppsFailedProcessingCount, tx.instance.Namespace)

	r.reportEvent(tx, corev1.EventTypeNormal, v1.EventRetrying, "Retrying synchronization")
	return ctrl.Result{Requeue: true}, nil
}

func (r *Reconciler) complete(tx transaction) (ctrl.Result, error) {
	if err := r.updateStatus(tx); err != nil {
		r.reportEvent(tx, corev1.EventTypeWarning, v1.EventFailedStatusUpdate, "Failed to update status")
		return ctrl.Result{}, err
	}

	metrics.IncWithNamespaceLabel(metrics.AzureAppsProcessedCount, tx.instance.Namespace)
	r.reportEvent(tx, corev1.EventTypeNormal, v1.EventSynchronized, "Azure application is up-to-date")
	return ctrl.Result{}, nil
}

func (r *Reconciler) updateStatus(tx transaction) error {
	tx.instance.Status.SynchronizationSecretName = tx.instance.Spec.SecretName
	tx.instance.Status.SynchronizationState = v1.EventSynchronized
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

	err = r.updateApplication(tx.ctx, tx.instance, func(existing *v1.AzureAdApplication) error {
		existing.Spec = tx.instance.Spec
		return r.Update(tx.ctx, existing)
	})
	if err != nil {
		return fmt.Errorf("updating spec: %w", err)
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
