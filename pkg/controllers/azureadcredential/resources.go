package azureadcredential

import (
	"context"
	"fmt"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/resourcecreator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) createOrUpdateResource(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential, creator resourcecreator.Creator) error {
	spec, err := creator.CreateSpec()
	mutateFn, err := creator.CreateMutateFn(spec)

	if err := ctrl.SetControllerReference(credential, spec.(metav1.Object), r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference %w", err)
	}

	res, err := ctrl.CreateOrUpdate(ctx, r.Client, spec, mutateFn)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("resource has been %s", res))
	return nil
}

func (r *Reconciler) createOrUpdateSecret(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential, application azure.Application) error {
	if err := r.createOrUpdateResource(ctx, credential, resourcecreator.Creator{
		Credential:  *credential,
		Application: application,
		Resource:    &corev1.Secret{},
	}); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) createOrUpdateConfigMap(ctx context.Context, credential *naisiov1alpha1.AzureAdCredential, application azure.Application) error {
	if err := r.createOrUpdateResource(ctx, credential, resourcecreator.Creator{
		Credential:  *credential,
		Application: application,
		Resource:    &corev1.ConfigMap{},
	}); err != nil {
		return err
	}
	return nil
}
