package fake

import (
	"github.com/google/uuid"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/transaction"
)

func ServicePrincipal(tx transaction.Transaction) msgraph.ServicePrincipal {
	id := uuid.New().String()
	return msgraph.ServicePrincipal{
		DirectoryObject: msgraph.DirectoryObject{Entity: msgraph.Entity{ID: &id}},
		DisplayName:     ptr.String(kubernetes.UniformResourceName(&tx.Instance, tx.ClusterName)),
	}
}
