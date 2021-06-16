package fake

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/azure/transaction"
)

type fakeAzureClient struct{}

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
		app := MsGraphApplication(tx.Instance)
		return &app, true, nil
	}
	return nil, false, nil
}

func (a fakeAzureClient) Get(tx transaction.Transaction) (msgraph.Application, error) {
	return MsGraphApplication(tx.Instance), nil
}

func (a fakeAzureClient) GetServicePrincipal(tx transaction.Transaction) (msgraph.ServicePrincipal, error) {
	return ServicePrincipal(tx.Instance), nil
}

func (a fakeAzureClient) GetPreAuthorizedApps(tx transaction.Transaction) (*result.PreAuthorizedApps, error) {
	return AzurePreAuthorizedApps(tx.Instance), nil
}

func (a fakeAzureClient) AddCredentials(tx transaction.Transaction) (credentials.Set, error) {
	return AzureCredentialsSet(tx.Instance), nil
}

func (a fakeAzureClient) RotateCredentials(tx transaction.Transaction, existing credentials.Set, inUse credentials.KeyIdsInUse) (credentials.Set, error) {
	newSet := AzureCredentialsSet(tx.Instance)
	newSet.Current = existing.Next
	return newSet, nil
}

func (a fakeAzureClient) PurgeCredentials(tx transaction.Transaction) error {
	return nil
}

func (a fakeAzureClient) ValidateCredentials(tx transaction.Transaction, existing credentials.Set) (bool, error) {
	return true, nil
}

func (a fakeAzureClient) Update(tx transaction.Transaction) (*result.Application, error) {
	internalApp := AzureApplicationResult(tx.Instance, result.OperationUpdated)
	return &internalApp, nil
}

func NewFakeAzureClient() azure.Client {
	return fakeAzureClient{}
}
