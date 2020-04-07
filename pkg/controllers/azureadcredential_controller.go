package controllers

import (
	"context"
	"fmt"

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

var log logr.Logger

// +kubebuilder:rbac:groups=nais.io,resources=azureadcredentials,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=azureadcredentials/status,verbs=get;update;patch

func (r *AzureAdCredentialReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log = r.Log.WithValues("azureadcredential", req.NamespacedName)

	var azureAdCredential naisiov1alpha1.AzureAdCredential
	if err := r.Get(ctx, req.NamespacedName, &azureAdCredential); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	azureAdCredentialHash, err := azureAdCredential.Hash()
	if err != nil {
		return ctrl.Result{Requeue: true}, fmt.Errorf("failed to calculate application hash: %w", err)
	}

	if azureAdCredential.Status.ProvisionHash == azureAdCredentialHash {
		log.Info("object state already reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	log.Info("processing AzureAdCredential", "azureAdCredential", azureAdCredential)
	return r.processAzureApplication(&ctx, &azureAdCredential)
}

func (r *AzureAdCredentialReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisiov1alpha1.AzureAdCredential{}).
		Complete(r)
}

func (r *AzureAdCredentialReconciler) processAzureApplication(ctx *context.Context, credential *naisiov1alpha1.AzureAdCredential) (ctrl.Result, error) {
	// Register or update (if exists) Azure application
	application, err := r.AzureClient.RegisterOrUpdateApplication(*credential)
	if err != nil {
		log.Error(err, "failed to register application")
		credential.StatusRetrying()
		_ = r.Status().Update(*ctx, credential)
		return ctrl.Result{Requeue: true}, nil
	}
	log.Info("successfully registered application", "clientId", application.ClientId)

	// Update AzureAdCredential.Status
	credential.StatusProvisioned()
	credential.SetCertificateKeyId(application.CertificateKeyId)
	credential.SetPasswordKeyId(application.PasswordKeyId)
	credential.SetClientId(application.ClientId)
	credential.SetObjectId(application.ObjectId)

	// Calculate and set new AzureAdCredential.Status.ProvisionHash
	credentialHash, err := credential.Hash()
	if err != nil {
		return ctrl.Result{Requeue: true}, fmt.Errorf("failed to calculate application hash: %w", err)
	}
	credential.Status.ProvisionHash = credentialHash

	// Update Status subresource
	if err := r.Status().Update(*ctx, credential); err != nil {
		log.Error(err, "could not update status for AzureAdCredential")
		credential.StatusRetrying()
		_ = r.Status().Update(*ctx, credential)
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

func (r *AzureAdCredentialReconciler) deleteAzureApplication(credential *naisiov1alpha1.AzureAdCredential) (ctrl.Result, error) {
	if err := r.AzureClient.DeleteApplication(*credential); err != nil {
		log.Error(err, "could not delete application in AzureAdCredential")
		return ctrl.Result{Requeue: true}, nil
	}
	log.Info("AzureAdCredential was deleted")
	return ctrl.Result{}, nil
}
