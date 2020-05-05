package azureadcredential

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AzureAdCredentialReconciler reconciles a AzureAdCredential object
type Reconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	AzureClient azure.Client
	ClusterName string
}

var log logr.Logger

// +kubebuilder:rbac:groups=nais.io,resources=azureadcredentials,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=azureadcredentials/status,verbs=get;update;patch

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log = r.Log.WithValues("azureadcredential", req.NamespacedName)
	ctx := context.Background()

	var azureAdCredential naisiov1alpha1.AzureAdCredential
	if err := r.Get(ctx, req.NamespacedName, &azureAdCredential); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	azureAdCredential.SetClusterName(r.ClusterName)
	log.Info("processing AzureAdCredential...", "azureAdCredential", azureAdCredential)

	// examine DeletionTimestamp to determine if object is under deletion
	if azureAdCredential.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, register finalizer if it doesn't exist
		if err := r.registerFinalizer(ctx, &azureAdCredential); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to register finalizer: %w", err)
		}
	} else {
		// The object is being deleted
		if err := r.processFinalizer(ctx, &azureAdCredential); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to process finalizer: %w", err)
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	hashUnchanged, err := azureAdCredential.HashUnchanged()
	if err != nil {
		return ctrl.Result{}, err
	}

	// TODO - should also check ProvisionState
	if hashUnchanged {
		log.Info("object state already reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	if err := r.process(ctx, &azureAdCredential); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to process Azure application: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisiov1alpha1.AzureAdCredential{}).
		Complete(r)
}

func (r *Reconciler) process(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	application, err := r.createOrUpdate(ctx, credential)
	if err != nil {
		credential.SetStatusRetrying()
		if err := r.updateStatusSubresource(ctx, credential); err != nil {
			return err
		}
		return err
	}
	log.Info("successfully synchronized AzureAdCredential with Azure")
	if err := r.updateStatus(ctx, credential, application); err != nil {
		return err
	}
	if err := r.createOrUpdateSecret(ctx, *credential, application); err != nil {
		return fmt.Errorf("failed to create or update secret: %w", err)
	}
	if err := r.createOrUpdateConfigMap(ctx, *credential, application); err != nil {
		return fmt.Errorf("failed to create or update configMap: %w", err)
	}
	return nil
}

func (r *Reconciler) createOrUpdate(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) (azure.Application, error) {
	var application azure.Application

	exists, err := r.AzureClient.Exists(ctx, *credential)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to lookup existence of application: %w", err)
	}

	if exists {
		application, err = r.update(ctx, credential)
		if err != nil {
			return azure.Application{}, fmt.Errorf("failed to update azure application: %w", err)
		}
	} else {
		application, err = r.create(ctx, credential)
		if err != nil {
			return azure.Application{}, fmt.Errorf("failed to create azure application: %w", err)
		}
	}
	return application, nil
}

// Update AzureAdCredential.Status
func (r *Reconciler) updateStatus(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential, application azure.Application) error {
	credential.SetCertificateKeyId(application.CertificateKeyId)
	credential.SetPasswordKeyId(application.PasswordKeyId)
	credential.SetClientId(application.ClientId)
	credential.SetApplicationObjectId(application.ObjectId)
	credential.SetStatusProvisioned()

	if err := credential.CalculateAndSetHash(); err != nil {
		return err
	}
	if err := r.updateStatusSubresource(ctx, credential); err != nil {
		return err
	}
	log.Info("status subresource successfully updated", "AzureAdCredentialStatus", credential.Status)
	return nil
}
