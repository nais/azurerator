package azureadapplication

import (
	"fmt"
	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/secrets"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
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

func ensurePreAuthAppsAreValid(req ctrl.Request, instance *v1.AzureAdApplication, clusterName string) {
	seen := map[string]v1.AccessPolicyRule{}

	preAuthApps := make([]v1.AccessPolicyRule, 0)

	for _, preAuthApp := range instance.Spec.PreAuthorizedApplications {
		if len(preAuthApp.Namespace) == 0 {
			preAuthApp.Namespace = req.Namespace
		}

		if len(preAuthApp.Cluster) == 0 {
			preAuthApp.Cluster = clusterName
		}

		name := preAuthApp.GetUniqueName()
		if _, found := seen[name]; !found {
			seen[name] = preAuthApp
			preAuthApps = append(preAuthApps, preAuthApp)
		}
	}

	instance.Spec.PreAuthorizedApplications = preAuthApps
}

func ensureReplyUrlsAreValid(instance *v1.AzureAdApplication) {
	seen := map[string]bool{}

	replyUrls := make([]v1.AzureAdReplyUrl, 0)

	for _, replyUrl := range instance.Spec.ReplyUrls {
		url := replyUrl.Url
		if _, found := seen[url]; !found {
			seen[url] = true
			replyUrls = append(replyUrls, replyUrl)
		}
	}

	instance.Spec.ReplyUrls = replyUrls
}

func ensureGroupClaimsAreValid(instance *v1.AzureAdApplication) {
	if instance.Spec.Claims == nil || len(instance.Spec.Claims.Groups) == 0 {
		return
	}

	seen := map[string]bool{}
	groups := make([]v1.AzureAdGroup, 0)

	for _, group := range instance.Spec.Claims.Groups {
		groupId := group.ID
		if _, found := seen[groupId]; !found {
			seen[groupId] = true
			groups = append(groups, group)
		}
	}

	instance.Spec.Claims.Groups = groups
}

func ensureExtraClaimsAreValid(instance *v1.AzureAdApplication) {
	if instance.Spec.Claims == nil || len(instance.Spec.Claims.Extra) == 0 {
		return
	}

	seen := map[v1.AzureAdExtraClaim]bool{}
	claims := make([]v1.AzureAdExtraClaim, 0)

	for _, claim := range instance.Spec.Claims.Extra {
		if _, found := seen[claim]; !found {
			seen[claim] = true
			claims = append(claims, claim)
		}
	}

	instance.Spec.Claims.Extra = claims
}
