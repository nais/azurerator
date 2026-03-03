package msgraph

import (
	"github.com/google/uuid"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/transaction"
)

func ServicePrincipal(tx transaction.Transaction) msgraph.ServicePrincipal {
	id := uuid.New().String()
	return msgraph.ServicePrincipal{
		DirectoryObject: msgraph.DirectoryObject{Entity: msgraph.Entity{ID: &id}},
		DisplayName:     new(tx.UniformResourceName),
	}
}
