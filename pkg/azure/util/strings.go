package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/azure"
)

func GetReplyUrlsStringSlice(resource v1.AzureAdApplication) []string {
	var replyUrls []string
	for _, v := range resource.Spec.ReplyUrls {
		replyUrls = append(replyUrls, v.Url)
	}
	return replyUrls
}

func IdentifierUri(id azure.ClientId) azure.IdentifierUri {
	return fmt.Sprintf("api://%s", id)
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
	return fmt.Sprintf("azurerator-%s", t.UTC().Format(time.RFC3339))
}
