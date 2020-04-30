package util

import (
	"fmt"

	"github.com/nais/azureator/pkg/apis/v1alpha1"
)

func GetReplyUrlsStringSlice(credential v1alpha1.AzureAdCredential) []string {
	var replyUrls []string
	for _, v := range credential.Spec.ReplyUrls {
		replyUrls = append(replyUrls, v.Url)
	}
	return replyUrls
}

func IdentifierUri(id string) string {
	return fmt.Sprintf("api://%s", id)
}
