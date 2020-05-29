package azureadapplication

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/metrics"
	"github.com/nais/azureator/pkg/secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AzureAdApplicationReconciler reconciles a AzureAdApplication object
type Reconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	AzureClient azure.Client
	Recorder    record.EventRecorder
	ClusterName string
}

type transaction struct {
	ctx      context.Context
	instance *v1alpha1.AzureAdApplication
	log      logr.Logger
}

func (t transaction) toAzureTx() azure.Transaction {
	return azure.Transaction{
		Ctx:      t.ctx,
		Instance: *t.instance,
		Log:      t.log,
	}
}

var log logr.Logger
var correlationId string

// +kubebuilder:rbac:groups=nais.io,resources=AzureAdApplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=AzureAdApplications/status,verbs=get;update;patch

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	correlationId = uuid.New().String()
	log = r.Log.WithValues("AzureAdApplication", req.NamespacedName, "correlationId", correlationId)
	ctx := context.Background()

	instance := &v1alpha1.AzureAdApplication{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	instance.SetClusterName(r.ClusterName)
	instance.Status.CorrelationId = correlationId
	log.Info("processing AzureAdApplication...")

	tx := transaction{ctx, instance, log}

	if instance.IsBeingDeleted() {
		if err := r.processFinalizer(tx); err != nil {
			return ctrl.Result{}, fmt.Errorf("error when processing finalizer: %v", err)
		}
		return ctrl.Result{}, nil
	}

	if !instance.HasFinalizer(FinalizerName) {
		if err := r.registerFinalizer(tx); err != nil {
			return ctrl.Result{}, fmt.Errorf("error when registering finalizer: %v", err)
		}
		return ctrl.Result{}, nil
	}

	upToDate, err := instance.IsUpToDate()
	if err != nil {
		return ctrl.Result{}, err
	}

	if upToDate {
		log.Info("object state already reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	if err := r.process(tx); err != nil {
		tx.instance.SetNotSynchronized()
		if err := r.updateStatusSubresource(tx); err != nil {
			return ctrl.Result{RequeueAfter: 10 * time.Second}, fmt.Errorf("failed to set synchronized status: %w", err)
		}
		r.Recorder.Event(tx.instance, corev1.EventTypeWarning, "Failed", "Failed to synchronize Azure application, retrying")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, fmt.Errorf("failed to process Azure application: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AzureAdApplication{}).
		Complete(r)
}

func (r *Reconciler) process(tx transaction) error {
	managedSecrets, err := r.getManagedSecrets(tx)
	if err != nil {
		return err
	}
	application, err := r.createOrUpdateAzureApp(tx, *managedSecrets)
	if err != nil {
		return err
	}
	if err := r.createOrUpdateSecret(tx, application); err != nil {
		return fmt.Errorf("failed to create or update secret: %w", err)
	}
	if err := r.updateStatus(tx, application); err != nil {
		return err
	}
	if err := r.deleteUnusedSecrets(tx, *managedSecrets); err != nil {
		return err
	}
	metrics.AzureAppsProcessedCount.Inc()
	r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Synchronized", "Azure application is up-to-date")
	return nil
}

func (r *Reconciler) createOrUpdateAzureApp(tx transaction, managedSecrets secret.Lists) (azure.Application, error) {
	var application *azure.Application

	exists, err := r.AzureClient.Exists(tx.toAzureTx())
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to lookup existence of application: %w", err)
	}

	if !exists {
		application, err = r.create(tx)
		if err != nil {
			return azure.Application{}, fmt.Errorf("failed to create azure application: %w", err)
		}
		r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Created", "Azure application is created")
	} else {
		application, err = r.update(tx)
		if err != nil {
			return azure.Application{}, fmt.Errorf("failed to update azure application: %w", err)
		}
		r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Updated", "Azure application is updated")

		application, err = r.rotate(tx, *application, managedSecrets)
		if err != nil {
			return azure.Application{}, fmt.Errorf("failed to rotate azure credentials: %w", err)
		}
		r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Rotated", "Azure credentials is rotated")
	}

	log.Info("successfully synchronized AzureAdApplication with Azure")
	return *application, nil
}

func (r *Reconciler) updateStatus(tx transaction, application azure.Application) error {
	log.Info("updating status for AzureAdApplication")
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
	log.Info("status subresource successfully updated",
		"CertificateKeyIDs", tx.instance.Status.CertificateKeyIds,
		"PasswordKeyIDs", tx.instance.Status.PasswordKeyIds,
		"ClientID", tx.instance.Status.ClientId,
		"ObjectID", tx.instance.Status.ObjectId,
		"ServicePrincipalID", tx.instance.Status.ServicePrincipalId,
	)
	return nil
}
