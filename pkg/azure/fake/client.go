package fake

import (
	"github.com/nais/azureator/pkg/azure"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type fakeAzureClient struct{}

const (
	ApplicationNotExistsName = "not-exists-in-azure"
	ApplicationExists        = "exists-in-azure"
)

func (a fakeAzureClient) Create(tx azure.Transaction) (*azure.ApplicationResult, error) {
	internalApp := InternalAzureApp(tx.Instance)
	return &internalApp, nil
}

func (a fakeAzureClient) Delete(azure.Transaction) error {
	return nil
}

func (a fakeAzureClient) Exists(tx azure.Transaction) (bool, error) {
	appExists := tx.Instance.Name == ApplicationExists
	validStatus := len(tx.Instance.GetObjectId()) > 0 && len(tx.Instance.GetClientId()) > 0
	if appExists || validStatus {
		return true, nil
	}
	return false, nil
}

func (a fakeAzureClient) Get(tx azure.Transaction) (msgraph.Application, error) {
	return ExternalAzureApp(tx.Instance), nil
}

func (a fakeAzureClient) GetServicePrincipal(tx azure.Transaction) (msgraphbeta.ServicePrincipal, error) {
	return ServicePrincipal(tx.Instance), nil
}

func (a fakeAzureClient) Rotate(tx azure.Transaction, app azure.ApplicationResult) (*azure.ApplicationResult, error) {
	internalApp := InternalAzureApp(tx.Instance)
	internalApp.Password.KeyId.AllInUse = append(app.Password.KeyId.AllInUse, internalApp.Password.KeyId.Latest)
	internalApp.Certificate.KeyId.AllInUse = append(app.Certificate.KeyId.AllInUse, internalApp.Certificate.KeyId.Latest)
	return &internalApp, nil
}

func (a fakeAzureClient) Update(tx azure.Transaction) (*azure.ApplicationResult, error) {
	internalApp := InternalAzureApp(tx.Instance)
	return &internalApp, nil
}

func NewFakeAzureClient() azure.Client {
	return fakeAzureClient{}
}
