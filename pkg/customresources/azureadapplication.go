package customresources

import (
	"fmt"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"time"
)

func HasExtraPolicy(in *nais_io_v1.AzureAdClaims, policyName nais_io_v1.AzureAdExtraClaim) bool {
	for _, policy := range in.Extra {
		if policy == policyName {
			return true
		}
	}
	return false
}

func IsHashChanged(in *nais_io_v1.AzureAdApplication) (bool, error) {
	newHash, err := in.Hash()
	if err != nil {
		return false, fmt.Errorf("calculating application hash: %w", err)
	}
	return in.Status.SynchronizationHash != newHash, nil
}

func IsSecretNameChanged(in *nais_io_v1.AzureAdApplication) bool {
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

func ShouldUpdateSecrets(in *nais_io_v1.AzureAdApplication, maxSecretAge time.Duration) bool {
	return IsSecretNameChanged(in) || HasExpiredSecrets(in, maxSecretAge)
}

func ShouldRotateSecrets(in *nais_io_v1.AzureAdApplication, maxSecretAge time.Duration) bool {
	secretNameChanged := IsSecretNameChanged(in)

	if in.Status.SynchronizationSecretRotationTime == nil {
		return secretNameChanged
	}

	if len(in.Status.SynchronizationSecretName) == 0 {
		return false
	}

	return secretNameChanged && HasExpiredSecrets(in, maxSecretAge)
}