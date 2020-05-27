package fixtures

import (
	"github.com/nais/azureator/pkg/azure"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type azureMockClient struct{}

const (
	ApplicationNotExistsName = "not-exists"
)

func (a azureMockClient) Create(tx azure.Transaction) (azure.Application, error) {
	return InternalAzureApp(tx.Instance), nil
}

func (a azureMockClient) Delete(azure.Transaction) error {
	return nil
}

func (a azureMockClient) Exists(tx azure.Transaction) (bool, error) {
	switch tx.Instance.Name {
	case ApplicationNotExistsName:
		return false, nil
	default:
		return true, nil
	}
}

func (a azureMockClient) Get(tx azure.Transaction) (msgraph.Application, error) {
	return ExternalAzureApp(tx.Instance), nil
}

func (a azureMockClient) GetServicePrincipal(tx azure.Transaction) (msgraphbeta.ServicePrincipal, error) {
	return ServicePrincipal(tx.Instance), nil
}

func (a azureMockClient) Rotate(tx azure.Transaction, app azure.Application) (azure.Application, error) {
	return InternalAzureApp(tx.Instance), nil
}

func (a azureMockClient) Update(tx azure.Transaction) (azure.Application, error) {
	return InternalAzureApp(tx.Instance), nil
}

func NewAzureClient() azure.Client {
	return azureMockClient{}
}
