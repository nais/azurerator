package azureadapplication

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/annotations"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

func (r *Reconciler) shouldSkip(tx *transaction) bool {
	if isNotInTeamNamespace(tx) {
		logger.Debug(fmt.Sprintf("Resource is annotated with '%s'. Skipping processing...", annotations.NotInTeamNamespaceKey))
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
			msg := fmt.Sprintf("ERROR: Expected resource in team namespace, but was found in namespace '%s'. Azure application and secrets will not be processed.", tx.instance.Namespace)
			logger.Error(msg)
			annotations.SetAnnotation(tx.instance, annotations.NotInTeamNamespaceKey, annotations.NotInTeamNamespaceValue)
			r.reportEvent(*tx, corev1.EventTypeWarning, v1.EventNotInTeamNamespace, msg)
			return true, nil
		}
	}
	return false, nil
}

func isNotInTeamNamespace(tx *transaction) bool {
	value, found := annotations.HasAnnotation(tx.instance, annotations.NotInTeamNamespaceKey)
	return found && (value == annotations.NotInTeamNamespaceValue)
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
