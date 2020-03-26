package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
)

// AzureAdCredentialReconciler reconciles a AzureAdCredential object
type AzureAdCredentialReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nais.io,resources=azureadcredentials,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=azureadcredentials/status,verbs=get;update;patch

func (r *AzureAdCredentialReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("azureadcredential", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *AzureAdCredentialReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisiov1alpha1.AzureAdCredential{}).
		Complete(r)
}
