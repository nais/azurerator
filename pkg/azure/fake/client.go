package fake

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/azure/transaction"
)

type fakeAzureClient struct{}

type fakeAzureCredentialsClient struct{}

const (
	ApplicationNotExistsName = "not-exists-in-azure"
	ApplicationExists        = "exists-in-azure"
)

func (a fakeAzureClient) Create(tx transaction.Transaction) (*result.Application, error) {
	internalApp := AzureApplicationResult(tx.Instance, result.OperationCreated)
	return &internalApp, nil
}

func (a fakeAzureClient) Delete(transaction.Transaction) error {
	return nil
}

func (a fakeAzureClient) Exists(tx transaction.Transaction) (*msgraph.Application, bool, error) {
	appExists := tx.Instance.Name == ApplicationExists
	validStatus := len(tx.Instance.GetObjectId()) > 0 && len(tx.Instance.GetClientId()) > 0
	if appExists || validStatus {
		app := MsGraphApplication(tx)
		return &app, true, nil
	}
	return nil, false, nil
}

func (a fakeAzureClient) Get(tx transaction.Transaction) (msgraph.Application, error) {
	return MsGraphApplication(tx), nil
}

func (a fakeAzureClient) GetServicePrincipal(tx transaction.Transaction) (msgraph.ServicePrincipal, error) {
	return ServicePrincipal(tx), nil
}

func (a fakeAzureClient) GetPreAuthorizedApps(tx transaction.Transaction) (*result.PreAuthorizedApps, error) {
	return AzurePreAuthorizedApps(tx.Instance), nil
}

func (a fakeAzureClient) Credentials() azure.Credentials {
	return fakeAzureCredentialsClient{}
}

func (a fakeAzureCredentialsClient) Add(tx transaction.Transaction) (credentials.Set, error) {
	return AzureCredentialsSet(tx.Instance, tx.ClusterName), nil
}

func (a fakeAzureCredentialsClient) DeleteExpired(tx transaction.Transaction) error {
	return nil
}

func (a fakeAzureCredentialsClient) DeleteUnused(tx transaction.Transaction, existing credentials.Set, keyIdsInUse credentials.KeyIdsInUse) error {
	return nil
}

func (a fakeAzureCredentialsClient) Rotate(tx transaction.Transaction, existing credentials.Set, inUse credentials.KeyIdsInUse) (credentials.Set, error) {
	newSet := AzureCredentialsSet(tx.Instance, tx.ClusterName)
	newSet.Current = existing.Next
	return newSet, nil
}

func (a fakeAzureCredentialsClient) Purge(tx transaction.Transaction) error {
	return nil
}

func (a fakeAzureCredentialsClient) Validate(tx transaction.Transaction, existing credentials.Set) (bool, error) {
	return true, nil
}

func (a fakeAzureClient) Update(tx transaction.Transaction) (*result.Application, error) {
	internalApp := AzureApplicationResult(tx.Instance, result.OperationUpdated)
	return &internalApp, nil
}

func NewFakeAzureClient() azure.Client {
	return fakeAzureClient{}
}
