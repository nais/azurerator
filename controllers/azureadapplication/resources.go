package azureadapplication

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/resourcecreator"
	"github.com/nais/azureator/pkg/secret"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) createOrUpdateResource(tx transaction, creator resourcecreator.Creator) (ctrlutil.OperationResult, error) {
	spec, err := creator.Spec()
	if err != nil {
		return ctrlutil.OperationResultNone, fmt.Errorf("could not create spec for instance: %w", err)
	}
	mutateFn, err := creator.MutateFn(spec)
	if err != nil {
		return ctrlutil.OperationResultNone, fmt.Errorf("could not create mutate function for instance: %w", err)
	}

	if err := ctrl.SetControllerReference(tx.instance, spec.(metav1.Object), r.Scheme); err != nil {
		return ctrlutil.OperationResultNone, fmt.Errorf("failed to set controller reference %w", err)
	}

	res, err := ctrl.CreateOrUpdate(tx.ctx, r.Client, spec, mutateFn)
	if err != nil {
		return ctrlutil.OperationResultNone, err
	}
	return res, nil
}

func (r *Reconciler) createOrUpdateSecret(tx transaction, application azure.Application) error {
	secretCreator := resourcecreator.NewSecret(*tx.instance, application)
	log.Info(fmt.Sprintf("processing secret with name '%s'...", secretCreator.Name()))
	res, err := r.createOrUpdateResource(tx, secretCreator)
	log.Info(fmt.Sprintf("secret %s", res))
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) getApplicationPods(tx transaction) (*corev1.PodList, error) {
	selector := client.MatchingLabels{
		resourcecreator.AppLabelKey: tx.instance.GetName(),
	}
	namespace := client.InNamespace(tx.instance.GetNamespace())
	podList := &corev1.PodList{}
	err := r.List(tx.ctx, podList, selector, namespace)
	if err != nil {
		return nil, err
	}
	return podList, nil
}

func (r *Reconciler) getSecrets(tx transaction) (corev1.SecretList, error) {
	var secrets corev1.SecretList
	var mLabels = client.MatchingLabels{}

	mLabels[resourcecreator.AppLabelKey] = tx.instance.GetName()
	mLabels[resourcecreator.TypeLabelKey] = resourcecreator.TypeLabelValue
	if err := r.List(tx.ctx, &secrets, client.InNamespace(tx.instance.Namespace), mLabels); err != nil {
		return secrets, err
	}
	return secrets, nil
}

func (r *Reconciler) getManagedSecrets(tx transaction) (*secret.Lists, error) {
	// fetch all application pods for this app
	podList, err := r.getApplicationPods(tx)
	if err != nil {
		return nil, err
	}

	// fetch all managed secrets
	allSecrets, err := r.getSecrets(tx)
	if err != nil {
		return nil, err
	}

	// find intersect between secrets in use by application pods and all managed secrets
	secrets := secret.PodSecretLists(allSecrets, *podList)
	return &secrets, nil
}

func (r *Reconciler) deleteUnusedSecrets(tx transaction, lists secret.Lists) error {
	for _, oldSecret := range lists.Unused.Items {
		log.Info(fmt.Sprintf("deleting unused secret '%s'...", oldSecret.Name))
		if err := r.Delete(tx.ctx, &oldSecret); err != nil {
			return fmt.Errorf("failed to delete unused secret: %w", err)
		}
	}
	return nil
}
