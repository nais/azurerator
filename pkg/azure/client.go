package azure

import (
	"github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/azure/transaction"
)

type Client interface {
	Create(tx transaction.Transaction) (*result.Application, error)
	Delete(tx transaction.Transaction) error
	Exists(tx transaction.Transaction) (*msgraph.Application, bool, error)
	Get(tx transaction.Transaction) (msgraph.Application, error)

	GetPreAuthorizedApps(tx transaction.Transaction) (*result.PreAuthorizedApps, error)
	GetServicePrincipal(tx transaction.Transaction) (msgraph.ServicePrincipal, error)

	AddCredentials(tx transaction.Transaction) (credentials.Set, error)
	DeleteUnusedCredentials(tx transaction.Transaction, existing credentials.Set, keyIdsInUse credentials.KeyIdsInUse) error
	PurgeCredentials(tx transaction.Transaction) error
	RotateCredentials(tx transaction.Transaction, existing credentials.Set, inUse credentials.KeyIdsInUse) (credentials.Set, error)
	ValidateCredentials(tx transaction.Transaction, existing credentials.Set) (bool, error)

	Update(tx transaction.Transaction) (*result.Application, error)
}
