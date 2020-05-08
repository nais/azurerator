package azureadapplication

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	naisiov1alpha1 "github.com/nais/azureator/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AzureAdApplicationReconciler reconciles a AzureAdApplication object
type Reconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	AzureClient azure.Client
	ClusterName string
}

type transaction struct {
	ctx      context.Context
	resource *naisiov1alpha1.AzureAdApplication
	log      logr.Logger
}

func (t transaction) toAzureTx() azure.Transaction {
	return azure.Transaction{
		Ctx:      t.ctx,
		Resource: *t.resource,
		Log:      t.log,
	}
}

var log logr.Logger

// +kubebuilder:rbac:groups=nais.io,resources=AzureAdApplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=AzureAdApplications/status,verbs=get;update;patch

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log = r.Log.WithValues("AzureAdApplication", req.NamespacedName)
	ctx := context.Background()

	var AzureAdApplication naisiov1alpha1.AzureAdApplication
	if err := r.Get(ctx, req.NamespacedName, &AzureAdApplication); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	AzureAdApplication.SetClusterName(r.ClusterName)
	log.Info("processing AzureAdApplication...", "AzureAdApplication", AzureAdApplication)

	tx := transaction{
		ctx,
		&AzureAdApplication,
		log,
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if AzureAdApplication.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, register finalizer if it doesn't exist
		if err := r.registerFinalizer(tx); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to register finalizer: %w", err)
		}
	} else {
		// The object is being deleted
		if err := r.processFinalizer(tx); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to process finalizer: %w", err)
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	hashUnchanged, err := AzureAdApplication.HashUnchanged()
	if err != nil {
		return ctrl.Result{}, err
	}

	if hashUnchanged && AzureAdApplication.Status.UpToDate {
		log.Info("object state already reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	if err := r.process(tx); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to process Azure application: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisiov1alpha1.AzureAdApplication{}).
		Complete(r)
}

func (r *Reconciler) process(tx transaction) error {
	application, err := r.createOrUpdateAzureApp(tx)
	if err != nil {
		tx.resource.SetStatusRetrying()
		if err := r.updateStatusSubresource(tx); err != nil {
			return err
		}
		return err
	}
	log.Info("successfully synchronized AzureAdApplication with Azure")
	if err := r.createOrUpdateSecret(tx, application); err != nil {
		return fmt.Errorf("failed to create or update secret: %w", err)
	}
	if err := r.createOrUpdateConfigMap(tx, application); err != nil {
		return fmt.Errorf("failed to create or update configMap: %w", err)
	}
	if err := r.updateStatus(tx, application); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) createOrUpdateAzureApp(tx transaction) (azure.Application, error) {
	var application azure.Application

	exists, err := r.AzureClient.Exists(tx.toAzureTx())
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to lookup existence of application: %w", err)
	}

	if exists {
		application, err = r.update(tx)
		if err != nil {
			return azure.Application{}, fmt.Errorf("failed to update azure application: %w", err)
		}
	} else {
		application, err = r.create(tx)
		if err != nil {
			return azure.Application{}, fmt.Errorf("failed to create azure application: %w", err)
		}
	}
	return application, nil
}

// Update AzureAdApplication.Status
func (r *Reconciler) updateStatus(tx transaction, application azure.Application) error {
	tx.resource.SetCertificateKeyId(application.CertificateKeyId)
	tx.resource.SetPasswordKeyId(application.PasswordKeyId)
	tx.resource.SetClientId(application.ClientId)
	tx.resource.SetObjectId(application.ObjectId)
	tx.resource.SetStatusProvisioned()

	if err := tx.resource.CalculateAndSetHash(); err != nil {
		return err
	}
	if err := r.updateStatusSubresource(tx); err != nil {
		return err
	}
	log.Info("status subresource successfully updated", "AzureAdApplicationStatus", tx.resource.Status)
	return nil
}
