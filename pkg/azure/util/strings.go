package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/nais/azureator/pkg/azure"
)

func MapFiltersToFilter(filters []azure.Filter) azure.Filter {
	if len(filters) > 0 {
		return strings.Join(filters[:], " ")
	} else {
		return ""
	}
}

func FilterByName(name azure.DisplayName) azure.Filter {
	return fmt.Sprintf("displayName eq '%s'", name)
}

func FilterByAppId(clientId azure.ClientId) azure.Filter {
	return fmt.Sprintf("appId eq '%s'", clientId)
}

func FilterByClientId(clientId azure.ClientId) azure.Filter {
	return fmt.Sprintf("clientId eq '%s'", clientId)
}

func DisplayName(t time.Time) azure.DisplayName {
	return fmt.Sprintf("%s-%s", azure.AzureratorPrefix, t.UTC().Format(time.RFC3339))
}
