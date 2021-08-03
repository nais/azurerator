package transaction

import (
	"context"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/azure/transaction"
	"github.com/nais/azureator/pkg/transaction/options"
	"github.com/nais/azureator/pkg/transaction/secrets"
)

type Transaction struct {
	Ctx      context.Context
	Instance *v1.AzureAdApplication
	Logger   log.Entry
	Options  options.TransactionOptions
	Secrets  secrets.Secrets
}

func (t *Transaction) ToAzureTx() transaction.Transaction {
	return transaction.Transaction{
		Ctx:      t.Ctx,
		Instance: *t.Instance,
		Log:      t.Logger,
	}
}
