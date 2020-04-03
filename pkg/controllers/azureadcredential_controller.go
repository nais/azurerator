package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	ctx := context.Background()
	r.Log.WithValues("azureadcredential", req.NamespacedName)

	var azureAdCredential naisiov1alpha1.AzureAdCredential
	if err := r.Get(ctx, req.NamespacedName, &azureAdCredential); err != nil {
		if errors.IsNotFound(err) {
			// todo: should clean up in Azure AD
			r.Log.Info("AzureAdCredential was deleted", "namespace/name", req.NamespacedName)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		r.Log.Error(err, "unable to fetch AzureAdCredential", "namespace/name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	var c = azureAdCredential.Status.Conditions
	if len(c) > 0 {
		if lastCondition := c[len(c)-1]; lastCondition.Reconciled() {
			return ctrl.Result{}, nil
		}
	}

	r.Log.Info("processing AzureAdCredential", "azureAdCredential", azureAdCredential)
	azureAdCredential.Status.Conditions = append(azureAdCredential.Status.Conditions, naisiov1alpha1.Condition{
		Type:               naisiov1alpha1.Completed,
		Status:             naisiov1alpha1.True,
		Reason:             "Completed",
		Message:            "Successfully processed AzureAdCredential",
		LastHeartbeatTime:  azureAdCredential.ObjectMeta.CreationTimestamp,
		LastTransitionTime: metav1.Now(),
	})

	azureAdCredential.Status.SynchronizationTime = metav1.Now()
	if err := r.Client.Update(ctx, &azureAdCredential); err != nil {
		r.Log.Error(err, "could not update status for AzureAdCredential") // todo
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	return ctrl.Result{}, nil
}

func (r *AzureAdCredentialReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisiov1alpha1.AzureAdCredential{}).
		Complete(r)
}
