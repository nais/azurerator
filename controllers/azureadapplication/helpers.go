package azureadapplication

import (
	"fmt"
)

func (r *Reconciler) updateStatusSubresource(tx transaction) error {
	if err := r.Status().Update(tx.ctx, tx.instance); err != nil {
		return fmt.Errorf("failed to update status subresource: %w", err)
	}
	return nil
}

func (r *Reconciler) ensureStatusIsValid(tx transaction) error {
	if len(tx.instance.Status.ClientId) == 0 || len(tx.instance.Status.ObjectId) == 0 {
		application, err := r.AzureClient.Get(tx.toAzureTx())
		if err != nil {
			return fmt.Errorf("failed to find object or client ID: %w", err)
		}
		tx.instance.Status.ClientId = *application.AppID
		tx.instance.Status.ObjectId = *application.ID
	}
	if len(tx.instance.Status.ServicePrincipalId) == 0 {
		sp, err := r.AzureClient.GetServicePrincipal(tx.toAzureTx())
		if err != nil {
			return fmt.Errorf("failed to get service principal for application: %w", err)
		}
		tx.instance.Status.ServicePrincipalId = *sp.ID
	}
	return nil
}

func (r *Reconciler) shouldSkip(tx *transaction) (bool, error) {
	_, found := tx.instance.ObjectMeta.Labels["skip"]
	if found {
		logger.Debugf("skip flag found on resource")
		return true, nil
	}

	namespaces, err := r.getSharedNamespaces(tx.ctx)
	if err != nil {
		return false, err
	}

	for _, ns := range namespaces.Items {
		if ns.Name == tx.instance.Namespace {
			logger.Debugf("resource exists in shared namespace '%s'", tx.instance.Namespace)
			tx.instance.SetSkipLabel()
			return true, nil
		}
	}
	return false, nil
}
