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
	hasResynchronizeAnnotation := customresources.HasResynchronizeAnnotation(instance)
	hasRotateAnnotation := customresources.HasRotateAnnotation(instance)
	hasNonExpiredSecrets := !customresources.HasExpiredSecrets(instance, b.Config.SecretRotation.MaxAge)
	tenantUnchanged := strings.Contains(instance.Status.SynchronizationTenant, b.Config.Azure.Tenant.Name)

	needsSynchronization := hashChanged || secretNameChanged || hasExpiredSecrets || hasResynchronizeAnnotation || hasRotateAnnotation
	needsAzureSynchronization := hashChanged || hasResynchronizeAnnotation
	hasValidSecrets := hasNonExpiredSecrets && tenantUnchanged
	needsSecretRotation := secretNameChanged || hasRotateAnnotation

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