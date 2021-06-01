package reconciler

import (
	"context"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/options"
	"github.com/nais/azureator/pkg/secrets"
)

type Transaction struct {
	Ctx      context.Context
	Instance *v1.AzureAdApplication
	Logger   log.Entry
	Options  options.TransactionOptions
	Secrets  TransactionSecrets
}

func (t *Transaction) ToAzureTx() azure.Transaction {
	return azure.Transaction{
		Ctx:      t.Ctx,
		Instance: *t.Instance,
		Log:      t.Logger,
	}
}

type TransactionSecrets struct {
	Credentials    TransactionCredentials
	DataKeys       secrets.SecretDataKeys
	KeyIdsInUse    azure.KeyIdsInUse
	ManagedSecrets kubernetes.SecretLists
}

type TransactionCredentials struct {
	Set   *azure.CredentialsSet
	Valid bool
}
