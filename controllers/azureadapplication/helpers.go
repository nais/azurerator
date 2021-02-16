package azureadapplication

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/secrets"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

func (r *Reconciler) shouldSkip(tx *transaction) bool {
	if hasSkipFlag(tx) {
		msg := fmt.Sprintf("Resource contains '%s' annotation. Skipping processing...", annotations.SkipKey)
		logger.Debug(msg)
		r.reportEvent(*tx, corev1.EventTypeWarning, v1.EventSkipped, msg)
		return true
	}

	if r.shouldSkipForTenant(tx) {
		logger.Debugf("resource is not addressed to tenant '%s', ignoring...", r.Config.Azure.Tenant.Name)
		return true
	} else {
		logger.Debugf("resource is addressed to tenant '%s', processing...", r.Config.Azure.Tenant.Name)
		return false
	}
}

func (r *Reconciler) createOrUpdateSecrets(tx transaction, application azure.ApplicationResult) error {
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
	config := r.Config.Azure.Tenant.Name
	tenant := tx.instance.Spec.Tenant

	if len(tenant) > 0 {
		logger.Debugf("found tenant in spec '%s', comparing with configured value '%s'...", tenant, config)
		return tenant != config
	}

	tenantRequired := r.Config.Validations.Tenant.Required

	if tenantRequired {
		logger.Debugf("required tenant not found in spec, skipping...")
	}

	return tenantRequired
}

func (r *Reconciler) inSharedNamespace(tx *transaction) (bool, error) {
	sharedNs, err := kubernetes.ListSharedNamespaces(tx.ctx, r.Reader)
	if err != nil {
		return false, err
	}
	for _, ns := range sharedNs.Items {
		if ns.Name == tx.instance.Namespace {
			msg := fmt.Sprintf("Resource should not exist in shared namespace '%s'. Skipping...", tx.instance.Namespace)
			logger.Debug(msg)
			annotations.SetAnnotation(tx.instance, annotations.SkipKey, annotations.SkipValue)
			r.reportEvent(*tx, corev1.EventTypeWarning, v1.EventNotInTeamNamespace, msg)
			r.reportEvent(*tx, corev1.EventTypeWarning, v1.EventSkipped, msg)
			return true, nil
		}
	}
	return false, nil
}

func hasSkipFlag(tx *transaction) bool {
	_, found := annotations.HasAnnotation(tx.instance, annotations.SkipKey)
	return found
}

var appsync sync.Mutex

func (r *Reconciler) updateApplication(ctx context.Context, app *v1.AzureAdApplication, updateFunc func(existing *v1.AzureAdApplication) error) error {
	appsync.Lock()
	defer appsync.Unlock()

	existing := &v1.AzureAdApplication{}
	err := r.Reader.Get(ctx, client.ObjectKey{Namespace: app.Namespace, Name: app.Name}, existing)
	if err != nil {
		return fmt.Errorf("get newest version of AzureAdApplication: %s", err)
	}

	return updateFunc(existing)
}
