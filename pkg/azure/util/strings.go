package util

import (
	"fmt"
	"strings"

	"github.com/nais/azureator/apis/v1alpha1"
)

func GetReplyUrlsStringSlice(resource v1alpha1.AzureAdApplication) []string {
	var replyUrls []string
	for _, v := range resource.Spec.ReplyUrls {
		replyUrls = append(replyUrls, v.Url)
	}
	return replyUrls
}

func IdentifierUri(id string) string {
	return fmt.Sprintf("api://%s", id)
}

func MapFiltersToFilter(filters []string) string {
	if len(filters) > 0 {
		return strings.Join(filters[:], " ")
	} else {
		return ""
	}
}

func FilterByName(name string) string {
	return fmt.Sprintf("displayName eq '%s'", name)
}

func FilterByAppId(clientId string) string {
	return fmt.Sprintf("appId eq '%s'", clientId)
}

func FilterByClientId(clientId string) string {
	return fmt.Sprintf("clientId eq '%s'", clientId)
}
