package azureadapplication

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/config"
	"time"

	"github.com/google/uuid"
	"github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/metrics"
	"github.com/nais/azureator/pkg/secrets"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	requeueInterval = 10 * time.Second
	contextTimeout  = 1 * time.Minute
)

// AzureAdApplicationReconciler reconciles a AzureAdApplication object
type Reconciler struct {
	client.Client
	Reader      client.Reader
	Scheme      *runtime.Scheme
	AzureClient azure.Client
	Recorder    record.EventRecorder
	Config      *config.Config
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
		logger.Info("skipping processing of this resource")
		return ctrl.Result{}, nil
	}

	if tx.instance.IsBeingDeleted() {
		return r.finalizer().process(*tx)
	}

	if !tx.instance.HasFinalizer(FinalizerName) {
		return r.finalizer().register(*tx)
	}

	inSharedNamespace, err := r.inSharedNamespace(tx)
	if err != nil {
		return ctrl.Result{}, err
	}

	if inSharedNamespace {
		if err := r.Client.Update(tx.ctx, tx.instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update resource with skip flag: %w", err)
		}
		metrics.IncWithNamespaceLabel(metrics.AzureAppsSkippedCount, tx.instance.Namespace)
		return ctrl.Result{}, nil
	}

	if upToDate, err := tx.instance.IsUpToDate(); upToDate {
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("object state already reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	application, err := r.process(*tx)
	if err != nil {
		return r.handleError(*tx, err)
	}

	return r.complete(*tx, *application)
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.AzureAdApplication{}).
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
	logger.Info("processing AzureAdApplication...")
	return &transaction{ctx, instance, logger}, nil
}

func (r *Reconciler) process(tx transaction) (*azure.Application, error) {
	tx.instance.SetNotSynchronized()

	managedSecrets, err := secrets.GetManaged(tx.ctx, tx.instance, r.Reader)
	if err != nil {
		return nil, err
	}

	application, err := r.createOrUpdateAzureApp(tx, *managedSecrets)
	if err != nil {
		return nil, err
	}

	if err := r.createOrUpdateSecrets(tx, application); err != nil {
		return nil, err
	}

	if err := r.deleteUnusedSecrets(tx, managedSecrets.Unused); err != nil {
		return nil, err
	}

	return &application, nil
}

func (r *Reconciler) handleError(tx transaction, err error) (ctrl.Result, error) {
	logger.Error(fmt.Errorf("failed to process Azure application: %w", err))
	r.Recorder.Event(tx.instance, corev1.EventTypeWarning, "Failed", fmt.Sprintf("Failed to synchronize Azure application, retrying in %s", requeueInterval))
	metrics.IncWithNamespaceLabel(metrics.AzureAppsFailedProcessingCount, tx.instance.Namespace)

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *Reconciler) complete(tx transaction, application azure.Application) (ctrl.Result, error) {
	if err := r.updateStatus(tx, application); err != nil {
		return ctrl.Result{}, err
	}

	metrics.IncWithNamespaceLabel(metrics.AzureAppsProcessedCount, tx.instance.Namespace)
	r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Synchronized", "Azure application is up-to-date")
	logger.Info("successfully reconciled")

	return ctrl.Result{}, nil
}

func (r *Reconciler) createOrUpdateAzureApp(tx transaction, managedSecrets secrets.Lists) (azure.Application, error) {
	var application *azure.Application

	exists, err := r.AzureClient.Exists(tx.toAzureTx())
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to lookup existence of application: %w", err)
	}

	if !exists {
		application, err = r.azure().create(tx)
		if err != nil {
			return azure.Application{}, fmt.Errorf("failed to create azure application: %w", err)
		}
		metrics.IncWithNamespaceLabel(metrics.AzureAppsCreatedCount, tx.instance.Namespace)
		r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Created", "Azure application is created")
	} else {
		application, err = r.azure().update(tx)
		if err != nil {
			return azure.Application{}, fmt.Errorf("failed to update azure application: %w", err)
		}
		metrics.IncWithNamespaceLabel(metrics.AzureAppsUpdatedCount, tx.instance.Namespace)
		r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Updated", "Azure application is updated")

		application, err = r.azure().rotate(tx, *application, managedSecrets)
		if err != nil {
			return azure.Application{}, fmt.Errorf("failed to rotate azure credentials: %w", err)
		}
		metrics.IncWithNamespaceLabel(metrics.AzureAppsRotatedCount, tx.instance.Namespace)
		r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Rotated", "Azure credentials is rotated")
	}

	logger.Info("successfully synchronized AzureAdApplication with Azure")
	return *application, nil
}

func (r *Reconciler) updateStatus(tx transaction, application azure.Application) error {
	logger.Debug("updating status for AzureAdApplication")
	tx.instance.Status.CertificateKeyIds = application.Certificate.KeyId.AllInUse
	tx.instance.Status.PasswordKeyIds = application.Password.KeyId.AllInUse
	tx.instance.Status.ClientId = application.ClientId
	tx.instance.Status.ObjectId = application.ObjectId
	tx.instance.Status.ServicePrincipalId = application.ServicePrincipalId
	tx.instance.SetSynchronized()

	if err := tx.instance.UpdateHash(); err != nil {
		return err
	}
	if err := r.updateStatusSubresource(tx); err != nil {
		return err
	}
	logger.WithFields(
		log.Fields{
			"CertificateKeyIDs":  tx.instance.Status.CertificateKeyIds,
			"PasswordKeyIDs":     tx.instance.Status.PasswordKeyIds,
			"ClientID":           tx.instance.Status.ClientId,
			"ObjectID":           tx.instance.Status.ObjectId,
			"ServicePrincipalID": tx.instance.Status.ServicePrincipalId,
		}).Info("status subresource successfully updated")
	return nil
}
