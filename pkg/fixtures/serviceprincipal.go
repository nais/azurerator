package fixtures

import (
	"github.com/google/uuid"
	"github.com/nais/azureator/api/v1alpha1"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
)

func ServicePrincipal(instance v1alpha1.AzureAdApplication) msgraphbeta.ServicePrincipal {
	id := uuid.New().String()
	return msgraphbeta.ServicePrincipal{
		DirectoryObject: msgraphbeta.DirectoryObject{Entity: msgraphbeta.Entity{ID: &id}},
		DisplayName:     ptr.String(instance.GetUniqueName()),
	}
}
