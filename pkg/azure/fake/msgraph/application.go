package msgraph

import (
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/fake"
	"github.com/nais/azureator/pkg/azure/transaction"
)

func Application(tx transaction.Transaction) msgraph.Application {
	objectId := fake.GetOrGenerate(tx.Instance.GetObjectId())
	clientId := fake.GetOrGenerate(tx.Instance.GetClientId())

	return msgraph.Application{
		DirectoryObject: msgraph.DirectoryObject{
			Entity: msgraph.Entity{ID: ptr.String(objectId)},
		},
		DisplayName: ptr.String(tx.UniformResourceName),
		AppID:       ptr.String(clientId),
	}
}
