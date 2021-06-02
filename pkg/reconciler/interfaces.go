package reconciler

import (
	"context"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/azureator/pkg/azure"
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
	Process(tx Transaction) (*azure.ApplicationResult, error)
	Delete(tx Transaction) error

	AddCredentials(tx Transaction, keyIdsInUse azure.KeyIdsInUse) (*azure.CredentialsSet, azure.KeyIdsInUse, error)
	RotateCredentials(tx Transaction, existing azure.CredentialsSet, keyIdsInUse azure.KeyIdsInUse) (*azure.CredentialsSet, azure.KeyIdsInUse, error)
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
	Prepare(ctx context.Context, instance *v1.AzureAdApplication) (*TransactionSecrets, error)
	Process(tx Transaction, applicationResult *azure.ApplicationResult) error
}
