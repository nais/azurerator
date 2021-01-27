package azurerator

import (
	"fmt"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
)

func HasExtraPolicy(in *nais_io_v1.AzureAdClaims, policyName nais_io_v1.AzureAdExtraClaim) bool {
	for _, policy := range in.Extra {
		if policy == policyName {
			return true
		}
	}
	return false
}

func IsUpToDate(in *nais_io_v1.AzureAdApplication) (bool, error) {
	newHash, err := in.Hash()
	if err != nil {
		return false, fmt.Errorf("calculating application hash: %w", err)
	}
	return in.Status.SynchronizationHash == newHash, nil
}

func ShouldUpdateSecrets(in *nais_io_v1.AzureAdApplication) bool {
	return in.Status.SynchronizationSecretName != in.Spec.SecretName
}
