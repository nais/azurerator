package azure

import (
	"github.com/nais/azureator/pkg/azure"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type azureMockClient struct{}

const (
	ApplicationNotExistsName = "not-exists-in-azure"
	ApplicationExists        = "exists-in-azure"
)

func (a azureMockClient) Create(tx azure.Transaction) (*azure.Application, error) {
	internalApp := InternalAzureApp(tx.Instance)
	return &internalApp, nil
}

func (a azureMockClient) Delete(azure.Transaction) error {
	return nil
}

func (a azureMockClient) Exists(tx azure.Transaction) (bool, error) {
	appExists := tx.Instance.Name == ApplicationExists
	validStatus := len(tx.Instance.Status.ObjectId) > 0 && len(tx.Instance.Status.ClientId) > 0
	if appExists || validStatus {
		return true, nil
	}
	return false, nil
}

func (a azureMockClient) Get(tx azure.Transaction) (msgraph.Application, error) {
	return ExternalAzureApp(tx.Instance), nil
}

func (a azureMockClient) GetServicePrincipal(tx azure.Transaction) (msgraphbeta.ServicePrincipal, error) {
	return ServicePrincipal(tx.Instance), nil
}

func (a azureMockClient) Rotate(tx azure.Transaction, _ azure.Application) (*azure.Application, error) {
	internalApp := InternalAzureApp(tx.Instance)
	return &internalApp, nil
}

func (a azureMockClient) Update(tx azure.Transaction) (*azure.Application, error) {
	internalApp := InternalAzureApp(tx.Instance)
	return &internalApp, nil
}

func NewAzureClient() azure.Client {
	return azureMockClient{}
}
