package azure

import (
	"github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/transaction"
)

type Client interface {
	Create(tx transaction.Transaction) (*result.Application, error)
	Delete(tx transaction.Transaction) error
	Exists(tx transaction.Transaction) (*msgraph.Application, bool, error)
	Get(tx transaction.Transaction) (msgraph.Application, error)
	Update(tx transaction.Transaction) (*result.Application, error)

	Credentials() Credentials

	GetPreAuthorizedApps(tx transaction.Transaction) (*result.PreAuthorizedApps, error)
	GetServicePrincipal(tx transaction.Transaction) (msgraph.ServicePrincipal, error)
}

type Credentials interface {
	Add(tx transaction.Transaction) (credentials.Set, error)
	DeleteExpired(tx transaction.Transaction) error
	DeleteUnused(tx transaction.Transaction) error
	Purge(tx transaction.Transaction) error
	Rotate(tx transaction.Transaction) (credentials.Set, error)
	Validate(tx transaction.Transaction, existing credentials.Set) (bool, error)
}
