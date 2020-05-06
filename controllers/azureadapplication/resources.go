package azureadapplication

import (
	"context"
	"fmt"

	"github.com/nais/azureator/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/resourcecreator"
	"github.com/nais/azureator/pkg/resourcecreator/configmap"
	"github.com/nais/azureator/pkg/resourcecreator/secret"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) createOrUpdateResource(ctx context.Context, resource v1alpha1.AzureAdApplication, creator resourcecreator.Creator) (ctrlutil.OperationResult, error) {
	spec, err := creator.Spec()
	if err != nil {
		return ctrlutil.OperationResultNone, fmt.Errorf("could not create spec for resource: %w", err)
	}
	mutateFn, err := creator.MutateFn(spec)
	if err != nil {
		return ctrlutil.OperationResultNone, fmt.Errorf("could not create mutate function for resource: %w", err)
	}

	if err := ctrl.SetControllerReference(&resource, spec.(metav1.Object), r.Scheme); err != nil {
		return ctrlutil.OperationResultNone, fmt.Errorf("failed to set controller reference %w", err)
	}

	res, err := ctrl.CreateOrUpdate(ctx, r.Client, spec, mutateFn)
	if err != nil {
		return ctrlutil.OperationResultNone, err
	}
	return res, nil
}

func (r *Reconciler) createOrUpdateSecret(ctx context.Context, resource v1alpha1.AzureAdApplication, application azure.Application) error {
	secretCreator := secret.New(resource, application)
	res, err := r.createOrUpdateResource(ctx, resource, secretCreator)
	log.Info(fmt.Sprintf("secret %s", res))
	if err != nil {
		return err
	}
	return nil
}

// TODO - should this be available in all namespaces for other apps?
func (r *Reconciler) createOrUpdateConfigMap(ctx context.Context, resource v1alpha1.AzureAdApplication, application azure.Application) error {
	configMapCreator := configmap.New(resource, application)
	res, err := r.createOrUpdateResource(ctx, resource, configMapCreator)
	log.Info(fmt.Sprintf("configMap %s", res))
	if err != nil {
		return err
	}
	return nil
}
