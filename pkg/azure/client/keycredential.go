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
	keys, err := c.getKeyCredentialSetInUse(tx)
	if err != nil {
		return msgraph.KeyCredential{}, crypto.JwkPair{}, err
	}
	keyCredential, jwkPair, err := generateNewKeyCredentialFor(tx.Resource)
	if err != nil {
		return msgraph.KeyCredential{}, crypto.JwkPair{}, err
	}
	keys = append(keys, keyCredential)
	app := util.EmptyApplication().Keys(keys).Build()
	if err := c.updateApplication(tx.Ctx, tx.Resource.Status.ObjectId, app); err != nil {
		return msgraph.KeyCredential{}, crypto.JwkPair{}, fmt.Errorf("failed to update application with keycredential: %w", err)
	}
	return keyCredential, jwkPair, nil
}

// Returns a set containing the newest KeyCredential in use, or empty if none exist
func (c client) getKeyCredentialSetInUse(tx azure.Transaction) ([]msgraph.KeyCredential, error) {
	application, err := c.Get(tx)
	if err != nil {
		return nil, err
	}
	var newestCredential msgraph.KeyCredential
	for _, keyCredential := range application.KeyCredentials {
		if newestCredential.StartDateTime == nil {
			newestCredential = keyCredential
		}
		if keyCredential.StartDateTime.After(*newestCredential.StartDateTime) {
			newestCredential = keyCredential
		}
		if string(*keyCredential.KeyID) == tx.Resource.Status.CertificateKeyId {
			return []msgraph.KeyCredential{keyCredential}, nil
		}
	}
	if newestCredential.StartDateTime == nil {
		return make([]msgraph.KeyCredential, 0), nil
	}
	return []msgraph.KeyCredential{newestCredential}, nil
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
