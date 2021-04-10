package azureadapplication

import (
	"fmt"
	"github.com/nais/azureator/pkg/azure"
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

func (p preAuthorizedAppsBuilder) reportInvalidAsEvents() {
	for _, app := range p.Invalid {
		p.Recorder.Event(p.AzureAdApplication, corev1.EventTypeWarning, v1.EventSkipped, fmt.Sprintf("Pre-authorized app '%s' was not found in the Azure AD tenant (%s)", app.Name, p.Config.Azure.Tenant.String()))
	}
}

func (p preAuthorizedAppsBuilder) shouldRequeueSynchronization() bool {
	return len(p.Invalid) > 0
}
