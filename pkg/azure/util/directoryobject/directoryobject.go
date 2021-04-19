package directoryobject

import (
	"fmt"

	msgraph "github.com/nais/msgraph.go/v1.0"
)

type OwnerPayload struct {
	Content string `json:"@odata.id"`
}

func ToOwnerPayload(owner msgraph.DirectoryObject) OwnerPayload {
	return OwnerPayload{
		Content: fmt.Sprintf("https://graph.microsoft.com/v1.0/directoryObjects/%s", *owner.ID),
	}
}

// Difference returns the elements in `a` that aren't in `b`.
// Shamelessly stolen and modified from https://stackoverflow.com/a/45428032/11868133
func Difference(a, b []msgraph.DirectoryObject) []msgraph.DirectoryObject {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		key := *x.ID
		mb[key] = struct{}{}
	}
	diff := make([]msgraph.DirectoryObject, 0)
	for _, x := range a {
		key := *x.ID
		if _, found := mb[key]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
