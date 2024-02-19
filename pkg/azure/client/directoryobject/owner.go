package directoryobject

import (
	"fmt"
	"slices"

	"github.com/nais/azureator/pkg/azure"
	msgraph "github.com/nais/msgraph.go/v1.0"
)

type OwnerPayload struct {
	Content string `json:"@odata.id"`
}

func ToOwnerPayload(id azure.ServicePrincipalId) OwnerPayload {
	return OwnerPayload{
		Content: fmt.Sprintf("https://graph.microsoft.com/v1.0/directoryObjects/%s", id),
	}
}

func ContainsOwner(owners []msgraph.DirectoryObject, id azure.ServicePrincipalId) bool {
	return slices.ContainsFunc(owners, func(obj msgraph.DirectoryObject) bool {
		return obj.ID != nil && *obj.ID == id
	})
}
