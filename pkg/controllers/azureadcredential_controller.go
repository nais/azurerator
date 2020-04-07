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

const finalizer string = "finalizer.azurerator.nais.io"

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

	// examine DeletionTimestamp to determine if object is under deletion
	if azureAdCredential.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, register finalizer if it doesn't exist
		if err := r.registerFinalizer(&ctx, &azureAdCredential); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to register finalizer: %w", err)
		}
	} else {
		// The object is being deleted
		if err := r.processFinalizer(&ctx, &azureAdCredential); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to process finalizer: %w", err)
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	azureAdCredentialHash, err := azureAdCredential.Hash()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to calculate application hash: %w", err)
	}

	if azureAdCredential.Status.ProvisionHash == azureAdCredentialHash {
		log.Info("object state already reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	log.Info("processing AzureAdCredential...", "azureAdCredential", azureAdCredential)
	if err := r.processAzureApplication(&ctx, &azureAdCredential); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to process Azure application: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *AzureAdCredentialReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisiov1alpha1.AzureAdCredential{}).
		Complete(r)
}

func (r *AzureAdCredentialReconciler) registerOrUpdateAzureApplication(credential *naisiov1alpha1.AzureAdCredential) (azure.Application, error) {
	exists, err := r.AzureClient.ApplicationExists(*credential)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	if exists {
		log.Info("Azure application already exists, updating...")
		credential.StatusRotateProvisioning()
		return r.AzureClient.UpdateApplication(*credential)
	} else {
		log.Info("Azure application not found, registering...")
		credential.StatusNewProvisioning()
		return r.AzureClient.RegisterApplication(*credential)
	}
}

func (r *AzureAdCredentialReconciler) processAzureApplication(ctx *context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	application, err := r.registerOrUpdateAzureApplication(credential)
	if err != nil {
		credential.StatusRetrying()
		_ = r.Status().Update(*ctx, credential)
		return fmt.Errorf("failed to register/update Azure application: %w", err)
	}
	log.Info("Azure application successfully registered/updated", "clientId", application.ClientId)

	// Update AzureAdCredential.Status
	credential.SetCertificateKeyId(application.CertificateKeyId)
	credential.SetPasswordKeyId(application.PasswordKeyId)
	credential.SetClientId(application.ClientId)
	credential.SetObjectId(application.ObjectId)
	credential.StatusProvisioned()

	// Calculate and set new AzureAdCredential.Status.ProvisionHash
	credentialHash, err := credential.Hash()
	if err != nil {
		return fmt.Errorf("failed to calculate application hash: %w", err)
	}
	credential.Status.ProvisionHash = credentialHash

	// Update Status subresource
	if err := r.Status().Update(*ctx, credential); err != nil {
		return fmt.Errorf("failed to update status subresource: %w", err)
	}
	return nil
}

// Delete external resources
func (r *AzureAdCredentialReconciler) delete(credential *naisiov1alpha1.AzureAdCredential) error {
	if err := r.deleteAzureApplication(credential); err != nil {
		return err
	}
	return nil
}

func (r *AzureAdCredentialReconciler) deleteAzureApplication(credential *naisiov1alpha1.AzureAdCredential) error {
	log.Info("deleting Azure application...")
	if err := r.AzureClient.DeleteApplication(*credential); err != nil {
		return fmt.Errorf("failed to delete Azure application: %w", err)
	}
	log.Info("Azure application successfully deleted")
	return nil
}

func (r *AzureAdCredentialReconciler) registerFinalizer(ctx *context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	if !containsString(credential.ObjectMeta.Finalizers, finalizer) {
		log.Info("finalizer for object not found, registering...")
		credential.ObjectMeta.Finalizers = append(credential.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(*ctx, credential); err != nil {
			return err
		}
		log.Info("finalizer successfully registered")
	}
	return nil
}

func (r *AzureAdCredentialReconciler) processFinalizer(ctx *context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	if containsString(credential.ObjectMeta.Finalizers, finalizer) {
		log.Info("finalizer triggered, deleting resources...")
		// our finalizer is present, so lets handle any external dependency
		if err := r.delete(credential); err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}

		// remove our finalizer from the list and update it.
		credential.ObjectMeta.Finalizers = removeString(credential.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(*ctx, credential); err != nil {
			return fmt.Errorf("failed to remove finalizer from list: %w", err)
		}
	}
	log.Info("finalizer finished successfully")
	return nil
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
