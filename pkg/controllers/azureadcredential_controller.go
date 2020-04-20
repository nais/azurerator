package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/util"
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
	log = r.Log.WithValues("azureadcredential", req.NamespacedName)
	ctx := context.Background()

	var azureAdCredential naisiov1alpha1.AzureAdCredential
	if err := r.Get(ctx, req.NamespacedName, &azureAdCredential); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
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
	if hashUnchanged {
		log.Info("object state already reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	if err := r.processAzureApplication(ctx, &azureAdCredential); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to process Azure application: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *AzureAdCredentialReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisiov1alpha1.AzureAdCredential{}).
		Complete(r)
}

func (r *AzureAdCredentialReconciler) registerOrUpdateAzureApplication(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) (azure.Application, error) {
	exists, err := r.AzureClient.ApplicationExists(ctx, *credential)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to lookup existence of application: %w", err)
	}
	if exists {
		log.Info("Azure application already exists, updating...")
		credential.StatusRotateProvisioning()
		if err := r.updateStatusSubresource(ctx, credential); err != nil {
			return azure.Application{}, err
		}
		return r.AzureClient.UpdateApplication(ctx, *credential)
	} else {
		log.Info("Azure application not found, registering...")
		credential.StatusNewProvisioning()
		if err := r.updateStatusSubresource(ctx, credential); err != nil {
			return azure.Application{}, err
		}
		return r.AzureClient.RegisterApplication(ctx, *credential)
	}
}

func (r *AzureAdCredentialReconciler) processAzureApplication(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	application, err := r.registerOrUpdateAzureApplication(ctx, credential)
	if err != nil {
		credential.StatusRetrying()
		if err := r.updateStatusSubresource(ctx, credential); err != nil {
			return err
		}
		return fmt.Errorf("failed to register/update Azure application: %w", err)
	}
	log.Info("Azure application successfully registered/updated", "AzureApplication", application)

	// Update AzureAdCredential.Status
	credential.SetCertificateKeyId(application.CertificateKeyId)
	credential.SetPasswordKeyId(application.PasswordKeyId)
	credential.SetClientId(application.ClientId)
	credential.SetObjectId(application.ObjectId)
	credential.StatusProvisioned()

	if err := credential.UpdateHash(); err != nil {
		return err
	}
	if err := r.updateStatusSubresource(ctx, credential); err != nil {
		return err
	}
	log.Info("Status subresource successfully updated", "AzureAdCredentialStatus", credential.Status)
	return nil
}

func (r *AzureAdCredentialReconciler) updateStatusSubresource(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	if err := r.Status().Update(ctx, credential); err != nil {
		return fmt.Errorf("failed to update status subresource: %w", err)
	}
	return nil
}

// Delete external resources
func (r *AzureAdCredentialReconciler) delete(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	if err := r.deleteAzureApplication(ctx, credential); err != nil {
		return err
	}
	return nil
}

func (r *AzureAdCredentialReconciler) deleteAzureApplication(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	log.Info("deleting Azure application...")
	exists, err := r.AzureClient.ApplicationExists(ctx, *credential)
	if err != nil {
		return err
	}
	if !exists {
		log.Info("Azure application does not exist - skipping deletion")
		return nil
	}
	if err := r.AzureClient.DeleteApplication(ctx, *credential); err != nil {
		return fmt.Errorf("failed to delete Azure application: %w", err)
	}
	log.Info("Azure application successfully deleted")
	return nil
}

func (r *AzureAdCredentialReconciler) registerFinalizer(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	if !util.ContainsString(credential.ObjectMeta.Finalizers, finalizer) {
		log.Info("finalizer for object not found, registering...")
		credential.ObjectMeta.Finalizers = append(credential.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(ctx, credential); err != nil {
			return err
		}
		log.Info("finalizer successfully registered")
	}
	return nil
}

func (r *AzureAdCredentialReconciler) processFinalizer(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential) error {
	if util.ContainsString(credential.ObjectMeta.Finalizers, finalizer) {
		log.Info("finalizer triggered, deleting resources...")
		// our finalizer is present, so lets handle any external dependency
		if err := r.delete(ctx, credential); err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}

		// remove our finalizer from the list and update it.
		credential.ObjectMeta.Finalizers = util.RemoveString(credential.ObjectMeta.Finalizers, finalizer)
		if err := r.Update(ctx, credential); err != nil {
			return fmt.Errorf("failed to remove finalizer from list: %w", err)
		}
	}
	log.Info("finalizer finished successfully")
	return nil
}
