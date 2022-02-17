package options

import (
	"strings"

	"github.com/nais/azureator/pkg/customresources"
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
	hasExpiredSecrets := customresources.HasExpiredSecrets(instance, b.config.SecretRotation.MaxAge)
	tenantUnchanged := strings.Contains(instance.Status.SynchronizationTenant, b.config.Azure.Tenant.Id)

	needsSynchronization := hashChanged || secretNameChanged || hasExpiredSecrets || hasResynchronizeAnnotation || hasRotateAnnotation
	needsAzureSynchronization := hashChanged || hasResynchronizeAnnotation
	hasValidSecrets := !hasExpiredSecrets && tenantUnchanged && b.secrets.Credentials.Valid && b.secrets.Credentials.Set != nil
	needsSecretRotation := secretNameChanged || hasRotateAnnotation
	needsCleanup := !needsSecretRotation && !instance.Spec.SecretProtected && b.config.SecretRotation.Cleanup
	// TODO - when moving clusters, resync all apps first to ensure new secretname, then ensure SecretRotation.Cleanup is disabled

	return ProcessOptions{
		Synchronize: needsSynchronization,
		Azure: AzureOptions{
			Synchronize:    needsAzureSynchronization,
			CleanupOrphans: b.config.Azure.Features.CleanupOrphans.Enabled,
		},
		Secret: SecretOptions{
			Rotate:  needsSecretRotation,
			Valid:   hasValidSecrets,
			Cleanup: needsCleanup,
		},
	}, nil
}

type ProcessOptions struct {
	Synchronize bool
	Azure       AzureOptions
	Secret      SecretOptions
}

type AzureOptions struct {
	Synchronize    bool
	CleanupOrphans bool
}

type SecretOptions struct {
	Rotate  bool
	Valid   bool
	Cleanup bool
}
