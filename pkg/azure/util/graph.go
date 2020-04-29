package util

import (
	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

func GetReplyUrlsStringSlice(credential v1alpha1.AzureAdCredential) []string {
	var replyUrls []string
	for _, v := range credential.Spec.ReplyUrls {
		replyUrls = append(replyUrls, v.Url)
	}
	return replyUrls
}

// TODO - unique displayname?
func ToKeyCredential(jwkPair crypto.JwkPair) msgraph.KeyCredential {
	keyId := msgraph.UUID(uuid.New().String())
	keyBase64 := msgraph.Binary(jwkPair.PublicPem)
	return msgraph.KeyCredential{
		KeyID:       &keyId,
		DisplayName: ptr.String("azurerator"),
		Type:        ptr.String("AsymmetricX509Cert"),
		Usage:       ptr.String("Verify"),
		Key:         &keyBase64,
	}
}
