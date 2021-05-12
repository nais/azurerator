package fake

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
)

type fakeAzureClient struct{}

const (
	ApplicationNotExistsName = "not-exists-in-azure"
	ApplicationExists        = "exists-in-azure"
)

func (a fakeAzureClient) Create(tx azure.Transaction) (*azure.ApplicationResult, error) {
	internalApp := AzureApplicationResult(tx.Instance, azure.OperationResultCreated)
	return &internalApp, nil
}

func (a fakeAzureClient) Delete(azure.Transaction) error {
	return nil
}

func (a fakeAzureClient) Exists(tx azure.Transaction) (*msgraph.Application, bool, error) {
	appExists := tx.Instance.Name == ApplicationExists
	validStatus := len(tx.Instance.GetObjectId()) > 0 && len(tx.Instance.GetClientId()) > 0
	if appExists || validStatus {
		app := MsGraphApplication(tx.Instance)
		return &app, true, nil
	}
	return nil, false, nil
}

func (a fakeAzureClient) Get(tx azure.Transaction) (msgraph.Application, error) {
	return MsGraphApplication(tx.Instance), nil
}

func (a fakeAzureClient) GetServicePrincipal(tx azure.Transaction) (msgraph.ServicePrincipal, error) {
	return ServicePrincipal(tx.Instance), nil
}

func (a fakeAzureClient) GetPreAuthorizedApps(tx azure.Transaction) (*azure.PreAuthorizedApps, error) {
	return AzurePreAuthorizedApps(tx.Instance), nil
}

func (a fakeAzureClient) AddCredentials(tx azure.Transaction) (azure.CredentialsSet, error) {
	return AzureCredentialsSet(tx.Instance), nil
}

func (a fakeAzureClient) RotateCredentials(tx azure.Transaction, existing azure.CredentialsSet, inUse azure.KeyIdsInUse) (azure.CredentialsSet, error) {
	newSet := AzureCredentialsSet(tx.Instance)
	newSet.Current = existing.Next
	return newSet, nil
}

func (a fakeAzureClient) ValidateCredentials(tx azure.Transaction, existing azure.CredentialsSet) (bool, error) {
	return true, nil
}

func (a fakeAzureClient) Update(tx azure.Transaction) (*azure.ApplicationResult, error) {
	internalApp := AzureApplicationResult(tx.Instance, azure.OperationResultUpdated)
	return &internalApp, nil
}

func NewFakeAzureClient() azure.Client {
	return fakeAzureClient{}
}
