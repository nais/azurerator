package azureadapplication

import (
	"fmt"
	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/namespaces"
	"github.com/nais/azureator/pkg/secrets"
	corev1 "k8s.io/api/core/v1"
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

func (r *Reconciler) shouldSkip(tx *transaction) bool {
	if hasSkipFlag(tx) {
		msg := fmt.Sprintf("Resource contains '%s' annotation. Skipping processing...", annotations.SkipKey)
		logger.Debug(msg)
		r.Recorder.Event(tx.instance, corev1.EventTypeWarning, "Skipped", msg)
		return true
	}

	if r.shouldSkipForTenant(tx) {
		logger.Debugf("resource is not addressed to tenant '%s', ignoring...", r.Config.AzureAd.TenantName)
		return true
	} else {
		logger.Debugf("resource is addressed to tenant '%s', processing...", r.Config.AzureAd.TenantName)
		return false
	}
}

func (r *Reconciler) createOrUpdateSecrets(tx transaction, application azure.Application) error {
	logger.Infof("processing secret with name '%s'...", tx.instance.Spec.SecretName)
	res, err := secrets.CreateOrUpdate(tx.ctx, tx.instance, application, r.Client, r.Scheme)
	if err != nil {
		return fmt.Errorf("failed to create or update secret: %w", err)
	}
	logger.Infof("secret '%s' %s", tx.instance.Spec.SecretName, res)
	return nil
}

func (r *Reconciler) deleteUnusedSecrets(tx transaction, unused corev1.SecretList) error {
	for _, oldSecret := range unused.Items {
		if oldSecret.Name == tx.instance.Spec.SecretName {
			continue
		}
		logger.Infof("deleting unused secret '%s'...", oldSecret.Name)
		if err := secrets.Delete(tx.ctx, oldSecret, r.Client); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) shouldSkipForTenant(tx *transaction) bool {
	tenantName := r.Config.AzureAd.TenantName
	annotationRequired := r.Config.Annotations.Tenant.Required

	value, found := annotations.HasAnnotation(tx.instance, annotations.TenantKey)

	if found {
		logger.Debugf("found annotation '%s: %s', comparing with configured value '%s'...", annotations.TenantKey, value, tenantName)
		return tenantName != value
	}

	if annotationRequired {
		logger.Debugf("required annotation '%s' not found, skipping...", annotations.TenantKey)
	}
	return annotationRequired
}

func (r *Reconciler) inSharedNamespace(tx *transaction) (bool, error) {
	sharedNs, err := namespaces.GetShared(tx.ctx, r.Reader)
	if err != nil {
		return false, err
	}
	for _, ns := range sharedNs.Items {
		if ns.Name == tx.instance.Namespace {
			msg := fmt.Sprintf("Resource should not exist in shared namespace '%s'. Skipping...", tx.instance.Namespace)
			logger.Debug(msg)
			tx.instance.SetSkipAnnotation()
			r.Recorder.Event(tx.instance, corev1.EventTypeWarning, "Skipped", msg)
			return true, nil
		}
	}
	return false, nil
}

func hasSkipFlag(tx *transaction) bool {
	_, found := annotations.HasAnnotation(tx.instance, annotations.SkipKey)
	return found
}
