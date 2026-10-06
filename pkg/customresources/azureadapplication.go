package customresources

import (
	"fmt"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/azureator/pkg/annotations"
)

func IsHashChanged(in *nais_io_v1.AzureAdApplication) (bool, error) {
	newHash, err := in.Hash()
	if err != nil {
		return false, fmt.Errorf("calculating application hash: %w", err)
	}
	return in.Status.SynchronizationHash != newHash, nil
}

func SecretNameChanged(in *nais_io_v1.AzureAdApplication) bool {
	return in.Status.SynchronizationSecretName != in.Spec.SecretName
}

func HasExpiredSecrets(in *nais_io_v1.AzureAdApplication, maxSecretAge time.Duration) bool {
	if in.Status.SynchronizationSecretRotationTime == nil {
		return false
	}

	lastRotationTime := *in.Status.SynchronizationSecretRotationTime
	diff := time.Since(lastRotationTime.Time)
	secretExpired := diff >= maxSecretAge

	return secretExpired
}

// CredentialValidationDelay returns the time left of the grace period after the last credential rotation, or 0 outside it.
func CredentialValidationDelay(in *nais_io_v1.AzureAdApplication, grace time.Duration) time.Duration {
	rotationTime := in.Status.SynchronizationSecretRotationTime
	if rotationTime == nil {
		return 0
	}
	elapsed := time.Since(rotationTime.Time)
	if elapsed < 0 || elapsed >= grace {
		return 0
	}
	return grace - elapsed
}

func HasResynchronizeAnnotation(in *nais_io_v1.AzureAdApplication) bool {
	_, found := annotations.HasAnnotation(in, annotations.ResynchronizeKey)
	return found
}

func HasRotateAnnotation(in *nais_io_v1.AzureAdApplication) bool {
	_, found := annotations.HasAnnotation(in, annotations.RotateKey)
	return found
}
