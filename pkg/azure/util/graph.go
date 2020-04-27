package util

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"gopkg.in/square/go-jose.v2"
)

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

func CreateKeyCredential(jwk jose.JSONWebKey) msgraph.KeyCredential {
	keyId := msgraph.UUID(uuid.New().String())
	keyBase64 := msgraph.Binary(crypto.ConvertToPem(jwk.Certificates[0]))
	return msgraph.KeyCredential{
		KeyID:       &keyId,
		DisplayName: ptr.String("azurerator"),
		Type:        ptr.String("AsymmetricX509Cert"),
		Usage:       ptr.String("Verify"),
		Key:         &keyBase64,
	}
}

func GetReplyUrlsStringSlice(credential v1alpha1.AzureAdCredential) []string {
	var replyUrls []string
	for _, v := range credential.Spec.ReplyUrls {
		replyUrls = append(replyUrls, v.Url)
	}
	return replyUrls
}
