package transaction

import (
	"context"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/transaction/options"
	"github.com/nais/azureator/pkg/transaction/secrets"
)

type Transaction struct {
	Ctx                 context.Context
	ClusterName         string
	Instance            *v1.AzureAdApplication
	Logger              log.Entry
	Options             options.TransactionOptions
	Secrets             secrets.Secrets
	ID                  string
	UniformResourceName string
}

func (t Transaction) UpdateWithApplicationIDs(application msgraph.Application) Transaction {
	t.Instance.Status.ClientId = *application.AppID
	t.Instance.Status.ObjectId = *application.ID
	return t
}

func (t Transaction) UpdateWithServicePrincipalID(servicePrincipal msgraph.ServicePrincipal) Transaction {
	t.Instance.Status.ServicePrincipalId = *servicePrincipal.ID
	return t
}
