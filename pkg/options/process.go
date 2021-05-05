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
	hasResynchronizeAnnotation := customresources.HasResynchronizeAnnotation(instance)
	hasRotateAnnotation := customresources.HasRotateAnnotation(instance)
	hasNonExpiredSecrets := !customresources.HasExpiredSecrets(instance, b.Config.SecretRotation.MaxAge)
	tenantUnchanged := strings.Contains(instance.Status.SynchronizationTenant, b.Config.Azure.Tenant.Name)

	needsAzureSynchronization := hashChanged || hasResynchronizeAnnotation
	hasValidSecrets := hasNonExpiredSecrets && tenantUnchanged
	needsSecretRotation := secretNameChanged || hasRotateAnnotation

	return ProcessOptions{
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
	Azure  AzureOptions
	Secret SecretOptions
}

type AzureOptions struct {
	Synchronize bool
}

type SecretOptions struct {
	Rotate bool
	Valid  bool
}
