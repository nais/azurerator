package reconciler

import (
	"context"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/transaction"
	"github.com/nais/azureator/pkg/transaction/secrets"
)

type AzureAdApplication interface {
	Azure() Azure
	Finalizer() Finalizer
	Secrets() Secrets

	ReportEvent(tx transaction.Transaction, eventType, event, message string)
	UpdateApplication(ctx context.Context, app *v1.AzureAdApplication, updateFunc func(existing *v1.AzureAdApplication) error) error
}

type Azure interface {
	Exists(tx transaction.Transaction) (bool, error)
	Delete(tx transaction.Transaction) error
	Process(tx transaction.Transaction) (*result.Application, error)
	ProcessOrphaned(tx transaction.Transaction) error

	AddCredentials(tx transaction.Transaction) (*credentials.Set, credentials.KeyID, error)
	DeleteExpiredCredentials(tx transaction.Transaction) error
	DeleteUnusedCredentials(tx transaction.Transaction) error
	RotateCredentials(tx transaction.Transaction) (*credentials.Set, credentials.KeyID, error)
	PurgeCredentials(tx transaction.Transaction) error
	ValidateCredentials(tx transaction.Transaction) (bool, error)
}

type Finalizer interface {
	Process(tx transaction.Transaction) (processed bool, err error)
}

type Secrets interface {
	Prepare(ctx context.Context, instance *v1.AzureAdApplication) (*secrets.Secrets, error)
	Process(tx transaction.Transaction, applicationResult *result.Application) error
	DeleteUnused(tx transaction.Transaction) error
}
