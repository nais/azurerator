package customresources

import (
	"fmt"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/event"
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

func HasResynchronizeAnnotation(in *nais_io_v1.AzureAdApplication) bool {
	_, found := annotations.HasAnnotation(in, annotations.ResynchronizeKey)
	return found
}

func HasRotateAnnotation(in *nais_io_v1.AzureAdApplication) bool {
	_, found := annotations.HasAnnotation(in, annotations.RotateKey)
	return found
}

func HasMatchingPreAuthorizedApp(in nais_io_v1.AzureAdApplication, event event.Event) bool {
	for _, preAuthApp := range in.Spec.PreAuthorizedApplications {
		if len(preAuthApp.Namespace) == 0 {
			preAuthApp.Namespace = in.GetNamespace()
		}
		if len(preAuthApp.Cluster) == 0 {
			preAuthApp.Cluster = in.GetClusterName()
		}

		nameMatches := preAuthApp.Application == event.Application.Name
		namespaceMatches := preAuthApp.Namespace == event.Application.Namespace
		clusterMatches := preAuthApp.Cluster == event.Application.Cluster

		if nameMatches && namespaceMatches && clusterMatches {
			return true
		}
	}

	return false
}
