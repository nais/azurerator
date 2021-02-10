package fake

import (
	"github.com/google/uuid"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
)

func ServicePrincipal(instance v1.AzureAdApplication) msgraphbeta.ServicePrincipal {
	id := uuid.New().String()
	return msgraphbeta.ServicePrincipal{
		DirectoryObject: msgraphbeta.DirectoryObject{Entity: msgraphbeta.Entity{ID: &id}},
		DisplayName:     ptr.String(kubernetes.UniformResourceName(&instance)),
	}
}
