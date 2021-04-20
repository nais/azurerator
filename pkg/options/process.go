package options

import (
	"github.com/nais/azureator/pkg/customresources"
	"strings"
)

func (b optionsBuilder) Process() (ProcessOptions, error) {
	instance := &b.instance

	hashChanged, err := customresources.IsHashChanged(instance)
	if err != nil {
		return ProcessOptions{}, err
	}

	secretNameChanged := customresources.SecretNameChanged(instance)
	hasExpiredSecrets := customresources.HasExpiredSecrets(instance, b.Config.SecretRotation.MaxAge)
	shouldResynchronize := customresources.ShouldResynchronize(instance)
	hasNonExpiredSecrets := !customresources.HasExpiredSecrets(instance, b.Config.SecretRotation.MaxAge)
	tenantUnchanged := strings.Contains(instance.Status.SynchronizationTenant, b.Config.Azure.Tenant.Name)

	needsSynchronization := hashChanged || secretNameChanged || hasExpiredSecrets || shouldResynchronize
	needsAzureSynchronization := hashChanged || shouldResynchronize
	hasValidSecrets := hasNonExpiredSecrets && tenantUnchanged
	needsSecretRotation := secretNameChanged

	return ProcessOptions{
		Synchronize: needsSynchronization,
		Azure: AzureOptions{
			Synchronize: needsAzureSynchronization,
		},
		Secret: SecretOptions{
			Rotate: needsSecretRotation,
			Valid:  hasValidSecrets,
		},
	}, nil
}

type ProcessOptions struct {
	Synchronize bool
	Azure       AzureOptions
	Secret      SecretOptions
}

type AzureOptions struct {
	Synchronize bool
}

type SecretOptions struct {
	Rotate bool
	Valid  bool
}
