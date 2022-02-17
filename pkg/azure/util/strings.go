package util

import (
	"fmt"
	"strings"
	"time"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/transaction"
)

func IdentifierUriClientId(id azure.ClientId) string {
	return fmt.Sprintf("api://%s", id)
}

func IdentifierUriHumanReadable(spec v1.AzureAdApplication) string {
	return fmt.Sprintf("api://%s.%s.%s", spec.GetClusterName(), spec.GetNamespace(), spec.GetName())
}

func IdentifierUris(tx transaction.Transaction) azure.IdentifierUris {
	return []string{
		IdentifierUriClientId(tx.Instance.GetClientId()),
		IdentifierUriHumanReadable(tx.Instance),
	}
}

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
