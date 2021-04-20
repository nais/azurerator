package options

import (
	"github.com/nais/azureator/pkg/customresources"
	"strings"
)

func (b optionsBuilder) Secret() SecretOptions {
	secretNameChanged := customresources.SecretNameChanged(&b.instance)
	hasNonExpiredSecrets := !customresources.HasExpiredSecrets(&b.instance, b.Config.SecretRotation.MaxAge)
	tenantUnchanged := strings.Contains(b.instance.Status.SynchronizationTenant, b.Config.Azure.Tenant.Name)

	valid := hasNonExpiredSecrets && tenantUnchanged

	return SecretOptions{
		Rotate: secretNameChanged,
		Valid:  valid,
	}
}

type SecretOptions struct {
	Rotate bool
	Valid  bool
}
