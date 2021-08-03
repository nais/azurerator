package options

import (
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
)

type TenantOptions struct {
	Ignore bool
}

func (b optionsBuilder) Tenant() TenantOptions {
	notAddressedToTenant := IsNotAddressedToTenant(b.instance, b.config.Azure.Tenant.Name, b.config.Validations.Tenant.Required)

	return TenantOptions{
		Ignore: notAddressedToTenant,
	}
}

func IsNotAddressedToTenant(instance v1.AzureAdApplication, configuredTenant string, requireMatchingTenant bool) bool {
	tenant := instance.Spec.Tenant

	if len(tenant) > 0 {
		return tenant != configuredTenant
	}

	return requireMatchingTenant
}
