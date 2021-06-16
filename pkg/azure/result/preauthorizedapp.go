package result

import (
	"github.com/nais/azureator/pkg/azure/resource"
)

type PreAuthorizedApps struct {
	// Valid is the list of apps that either are or can be assigned to an application in Azure AD.
	Valid []resource.Resource `json:"valid"`
	// Invalid is the list of apps that cannot be assigned to the application in Azure AD (e.g. apps that do not exist).
	Invalid []resource.Resource `json:"invalid"`
}
