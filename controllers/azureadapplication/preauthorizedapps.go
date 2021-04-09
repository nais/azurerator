package azureadapplication

import (
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/customresources"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
)

type preAuthorizedAppsBuilder struct {
	Reconciler
	*v1.AzureAdApplication
	azure.PreAuthorizedApps
}

func (r Reconciler) preauthorizedapps(application *v1.AzureAdApplication, preAuthorizedApps azure.PreAuthorizedApps) preAuthorizedAppsBuilder {
	return preAuthorizedAppsBuilder{
		Reconciler:         r,
		AzureAdApplication: application,
		PreAuthorizedApps:  preAuthorizedApps,
	}
}

// filterInvalid filters out invalid (i.e. not found in Azure AD, and thus not assigned to this Azure AD application)
// pre-authorized apps from the provided AzureAdApplication resource.
// This modifies the AzureAdApplication Spec - ensure that the resource is updated in the cluster.
func (p preAuthorizedAppsBuilder) filterInvalid() preAuthorizedAppsBuilder {
	filtered := make([]v1.AccessPolicyRule, 0)

	for _, declared := range p.Spec.PreAuthorizedApplications {
		if p.isValid(declared) {
			filtered = append(filtered, declared)
		}
	}

	p.Spec.PreAuthorizedApplications = filtered

	return p
}

func (p preAuthorizedAppsBuilder) reportInvalidAsEvents() {
	for _, app := range p.Invalid {
		p.Recorder.Event(p.AzureAdApplication, corev1.EventTypeWarning, v1.EventSkipped, fmt.Sprintf("Pre-authorized app '%s' was not found in the Azure AD tenant (%s)", app.Name, p.Config.Azure.Tenant.String()))
	}
}

func (p preAuthorizedAppsBuilder) isValid(wanted v1.AccessPolicyRule) bool {
	for _, candidate := range p.Valid {
		if customresources.GetUniqueName(wanted) == candidate.Name {
			return true
		}
	}
	return false
}
