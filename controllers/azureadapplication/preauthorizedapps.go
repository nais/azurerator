package azureadapplication

import (
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
)

type preAuthorizedAppsBuilder struct {
	Reconciler
	transaction
	azure.PreAuthorizedApps
}

func (r Reconciler) preauthorizedapps(tx transaction, preAuthorizedApps azure.PreAuthorizedApps) preAuthorizedAppsBuilder {
	return preAuthorizedAppsBuilder{
		Reconciler:        r,
		transaction:       tx,
		PreAuthorizedApps: preAuthorizedApps,
	}
}

func (p preAuthorizedAppsBuilder) reportInvalidAsEvents() preAuthorizedAppsBuilder {
	for _, app := range p.Invalid {
		msg := fmt.Sprintf("Pre-authorized app '%s' was not found in the Azure AD tenant (%s)", app.Name, p.Config.Azure.Tenant.String())
		p.transaction.log.Warnf(msg)
		p.Recorder.Event(p.instance, corev1.EventTypeWarning, v1.EventSkipped, msg)
	}
	return p
}
