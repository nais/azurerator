package secrets

import (
	"github.com/nais/liberator/pkg/kubernetes"

	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/secrets"
)

type Secrets struct {
	Credentials    Credentials
	DataKeys       secrets.SecretDataKeys
	KeyIdsInUse    credentials.KeyIdsInUse
	ManagedSecrets kubernetes.SecretLists
}

type Credentials struct {
	Set   *credentials.Set
	Valid bool
}
