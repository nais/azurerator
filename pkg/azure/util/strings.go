package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/azure"
)

func GetReplyUrlsStringSlice(resource v1.AzureAdApplication) []string {
	replyUrls := make([]string, 0)
	for _, v := range resource.Spec.ReplyUrls {
		replyUrls = append(replyUrls, v.Url)
	}
	return replyUrls
}

func IdentifierUriClientId(id azure.ClientId) string {
	return fmt.Sprintf("api://%s", id)
}

func IdentifierUriHumanReadable(spec v1.AzureAdApplication) string {
	return fmt.Sprintf("api://%s.%s.%s", spec.GetName(), spec.GetNamespace(), spec.GetClusterName())
}

func IdentifierUris(tx azure.Transaction) azure.IdentifierUris {
	return []string{
		IdentifierUriClientId(tx.Instance.Status.ClientId),
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
