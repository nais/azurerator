package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/nais/azureator/pkg/azure"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
)

// AzureAdCredentialReconciler reconciles a AzureAdCredential object
type AzureAdCredentialReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	AzureClient azure.Client
}

// +kubebuilder:rbac:groups=nais.io,resources=azureadcredentials,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=azureadcredentials/status,verbs=get;update;patch

func (r *AzureAdCredentialReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	r.Log = r.Log.WithValues("azureadcredential", req.NamespacedName)

	var azureAdCredential naisiov1alpha1.AzureAdCredential
	if err := r.Get(ctx, req.NamespacedName, &azureAdCredential); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	azureAdCredentialHash, err := azureAdCredential.Hash()
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	if azureAdCredential.Status.ProvisionHash == azureAdCredentialHash {
		r.Log.Info("object state already reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	r.Log.Info("processing AzureAdCredential", "azureAdCredential", azureAdCredential)

	application, err := r.AzureClient.RegisterOrUpdateApplication(azureAdCredential)
	if err != nil {
		r.Log.Error(err, "failed to register application")
		azureAdCredential.Status = azureAdCredential.Status.Retrying()
		_ = r.Status().Update(ctx, &azureAdCredential)
		return ctrl.Result{Requeue: true}, nil
	}

	r.Log.Info("successfully registered application", "clientId", application.Credentials.Public.ClientId)

	azureAdCredential.Status = azureAdCredential.Status.Provisioned(naisiov1alpha1.Provision{
		AadCredentialSpec: &azureAdCredential.Spec,
		Hash:              azureAdCredentialHash,
	})

	if err := r.Status().Update(ctx, &azureAdCredential); err != nil {
		r.Log.Error(err, "could not update status for AzureAdCredential")
		azureAdCredential.Status = azureAdCredential.Status.Retrying()
		_ = r.Status().Update(ctx, &azureAdCredential)
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

func (r *AzureAdCredentialReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisiov1alpha1.AzureAdCredential{}).
		Complete(r)
}
