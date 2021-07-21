package secrets

import (
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/liberator/pkg/kubernetes"
)

type TransactionSecrets struct {
	Credentials    TransactionCredentials
	DataKeys       SecretDataKeys
	KeyIdsInUse    credentials.KeyIdsInUse
	ManagedSecrets kubernetes.SecretLists
}

type TransactionCredentials struct {
	Set   *credentials.Set
	Valid bool
}
