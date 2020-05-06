package util

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

func GenerateNewKeyCredentialFor(resource v1alpha1.AzureAdApplication) (msgraph.KeyCredential, crypto.JwkPair, error) {
	jwkPair, err := crypto.GenerateJwkPair(resource)
	if err != nil {
		return msgraph.KeyCredential{}, crypto.JwkPair{}, fmt.Errorf("failed to generate JWK pair for application: %w", err)
	}
	newKeyCredential := toKeyCredential(jwkPair)
	return newKeyCredential, jwkPair, nil
}

// TODO - unique displayname?
func toKeyCredential(jwkPair crypto.JwkPair) msgraph.KeyCredential {
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
