package fake

import (
	"github.com/google/uuid"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
)

func ServicePrincipal(instance v1.AzureAdApplication) msgraph.ServicePrincipal {
	id := uuid.New().String()
	return msgraph.ServicePrincipal{
		DirectoryObject: msgraph.DirectoryObject{Entity: msgraph.Entity{ID: &id}},
		DisplayName:     ptr.String(kubernetes.UniformResourceName(&instance)),
	}
}
