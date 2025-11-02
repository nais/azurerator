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
	if in.Status.SynchronizationSecretRotationTime == nil || in.Spec.SecretProtected {
		return false
	}

	lastRotationTime := *in.Status.SynchronizationSecretRotationTime
	diff := time.Since(lastRotationTime.Time)
	secretExpired := diff >= maxSecretAge

	return secretExpired
}

func HasResynchronizeAnnotation(in *nais_io_v1.AzureAdApplication) bool {
	_, found := annotations.HasAnnotation(in, annotations.ResynchronizeKey)
	return found
}

func HasRotateAnnotation(in *nais_io_v1.AzureAdApplication) bool {
	_, found := annotations.HasAnnotation(in, annotations.RotateKey)
	return found
}
