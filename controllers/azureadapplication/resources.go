package azureadapplication

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/resourcecreator"
	"github.com/nais/azureator/pkg/resourcecreator/configmap"
	"github.com/nais/azureator/pkg/resourcecreator/secret"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) createOrUpdateResource(tx transaction, creator resourcecreator.Creator) (ctrlutil.OperationResult, error) {
	spec, err := creator.Spec()
	if err != nil {
		return ctrlutil.OperationResultNone, fmt.Errorf("could not create spec for resource: %w", err)
	}
	mutateFn, err := creator.MutateFn(spec)
	if err != nil {
		return ctrlutil.OperationResultNone, fmt.Errorf("could not create mutate function for resource: %w", err)
	}

	if err := ctrl.SetControllerReference(tx.resource, spec.(metav1.Object), r.Scheme); err != nil {
		return ctrlutil.OperationResultNone, fmt.Errorf("failed to set controller reference %w", err)
	}

	res, err := ctrl.CreateOrUpdate(tx.ctx, r.Client, spec, mutateFn)
	if err != nil {
		return ctrlutil.OperationResultNone, err
	}
	return res, nil
}

func (r *Reconciler) createOrUpdateSecret(tx transaction, application azure.Application) error {
	secretCreator := secret.New(*tx.resource, application)
	log.Info(fmt.Sprintf("processing secret with name '%s'...", secretCreator.Name()))
	res, err := r.createOrUpdateResource(tx, secretCreator)
	log.Info(fmt.Sprintf("secret %s", res))
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) createOrUpdateConfigMap(tx transaction, application azure.Application) error {
	configMapCreator := configmap.New(*tx.resource, application)
	log.Info(fmt.Sprintf("processing configMap with name '%s'...", configMapCreator.Name()))
	res, err := r.createOrUpdateResource(tx, configMapCreator)
	log.Info(fmt.Sprintf("configMap %s", res))
	if err != nil {
		return err
	}
	return nil
}
