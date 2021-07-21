package reconciler

import (
	"context"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/secrets"
)

type AzureAdApplication interface {
	Azure() Azure
	Finalizer() Finalizer
	Namespace() Namespace
	Secrets() Secrets

	ReportEvent(tx Transaction, eventType, event, message string)
	UpdateApplication(ctx context.Context, app *v1.AzureAdApplication, updateFunc func(existing *v1.AzureAdApplication) error) error
}

type Azure interface {
	Exists(tx Transaction) (bool, error)
	Delete(tx Transaction) error
	Process(tx Transaction) (*result.Application, error)
	ProcessOrphaned(tx Transaction) error

	AddCredentials(tx Transaction, keyIdsInUse credentials.KeyIdsInUse) (*credentials.Set, credentials.KeyIdsInUse, error)
	RotateCredentials(tx Transaction, existing credentials.Set, keyIdsInUse credentials.KeyIdsInUse) (*credentials.Set, credentials.KeyIdsInUse, error)
	PurgeCredentials(tx Transaction) error
	ValidateCredentials(tx Transaction) (bool, error)
}

type Finalizer interface {
	Process(tx Transaction) (processed bool, err error)
}

type Namespace interface {
	Process(tx *Transaction) (bool, error)
}

type Secrets interface {
	Prepare(ctx context.Context, instance *v1.AzureAdApplication) (*secrets.TransactionSecrets, error)
	Process(tx Transaction, applicationResult *result.Application) error
}
