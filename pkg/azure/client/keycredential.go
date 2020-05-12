package client

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/nais/azureator/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Generates a new set of key credentials, removing any key not in use (as indicated by AzureAdApplication.Status.CertificateKeyId).
// There should always be two active keys available at any given time so that running applications are not interfered with.
func (c client) rotateKeyCredential(tx azure.Transaction) (msgraph.KeyCredential, crypto.JwkPair, error) {
	existingKeyCredential, err := c.getExistingKeyCredential(tx)
	if err != nil {
		return msgraph.KeyCredential{}, crypto.JwkPair{}, err
	}
	keyCredential, jwkPair, err := generateNewKeyCredentialFor(tx.Resource)
	if err != nil {
		return msgraph.KeyCredential{}, crypto.JwkPair{}, err
	}
	keys := []msgraph.KeyCredential{keyCredential, existingKeyCredential}
	app := util.EmptyApplication().Keys(keys).Build()
	if err := c.updateApplication(tx.Ctx, tx.Resource.Status.ObjectId, app); err != nil {
		return msgraph.KeyCredential{}, crypto.JwkPair{}, fmt.Errorf("failed to update application with keycredential: %w", err)
	}
	return keyCredential, jwkPair, nil
}

func (c client) getExistingKeyCredential(tx azure.Transaction) (msgraph.KeyCredential, error) {
	application, err := c.Get(tx)
	if err != nil {
		return msgraph.KeyCredential{}, err
	}
	newestCredential := application.KeyCredentials[0]
	for _, keyCredential := range application.KeyCredentials {
		if keyCredential.StartDateTime.After(*newestCredential.StartDateTime) {
			newestCredential = keyCredential
		}
		if string(*keyCredential.KeyID) == tx.Resource.Status.CertificateKeyId {
			return keyCredential, nil
		}
	}
	if len(application.KeyCredentials) > 0 {
		return newestCredential, nil
	}
	return msgraph.KeyCredential{}, fmt.Errorf("failed to find key credential for azure application")
}

func generateNewKeyCredentialFor(resource v1alpha1.AzureAdApplication) (msgraph.KeyCredential, crypto.JwkPair, error) {
	jwkPair, err := crypto.GenerateJwkPair(resource)
	if err != nil {
		return msgraph.KeyCredential{}, crypto.JwkPair{}, fmt.Errorf("failed to generate JWK pair for application: %w", err)
	}
	newKeyCredential := toKeyCredential(jwkPair)
	return newKeyCredential, jwkPair, nil
}

func toKeyCredential(jwkPair crypto.JwkPair) msgraph.KeyCredential {
	keyId := msgraph.UUID(uuid.New().String())
	keyBase64 := msgraph.Binary(jwkPair.PublicPem)
	return msgraph.KeyCredential{
		KeyID:       &keyId,
		DisplayName: ptr.String(util.DisplayName()),
		Type:        ptr.String("AsymmetricX509Cert"),
		Usage:       ptr.String("Verify"),
		Key:         &keyBase64,
	}
}
