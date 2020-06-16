package client

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type keyCredential struct {
	client
}

func (c client) keyCredential() keyCredential {
	return keyCredential{c}
}

// Generates a new set of key credentials, removing any key not in use (as indicated by AzureAdApplication.Status.CertificateKeyIds).
// With the exception of new applications, there should always be two active keys available at any given time so that running applications are not interfered with.
func (k keyCredential) rotate(tx azure.Transaction, keyIdsInUse []string) (*msgraph.KeyCredential, *crypto.JwkPair, error) {
	keysInUse, err := k.mapToKeyCredentials(tx, keyIdsInUse)
	if err != nil {
		return nil, nil, err
	}
	keyCredential, jwkPair, err := k.new(tx.Instance)
	if err != nil {
		return nil, nil, err
	}
	keysInUse = append(keysInUse, *keyCredential)
	app := util.EmptyApplication().Keys(keysInUse).Build()
	if err := k.application().update(tx.Ctx, tx.Instance.Status.ObjectId, app); err != nil {
		return nil, nil, fmt.Errorf("failed to update application with keycredential: %w", err)
	}
	return keyCredential, jwkPair, nil
}

// Maps a list of key IDs to a list of KeyCredentials
func (k keyCredential) mapToKeyCredentials(tx azure.Transaction, keyIdsInUse []string) ([]msgraph.KeyCredential, error) {
	application, err := k.Get(tx)
	if err != nil {
		return nil, err
	}
	keyCredentialsInUse := make([]msgraph.KeyCredential, 0)

	// Keep the newest registered credential in case the app already exists in Azure and is not referenced by resources in the cluster.
	// This case assumes the possibility of the Azure application being used in applications external to the cluster.
	// There should always be at least one keycredential registered for an application.
	var newestCredential msgraph.KeyCredential
	for _, keyCredential := range application.KeyCredentials {
		if keyCredentialInUse(keyCredential, keyIdsInUse) {
			keyCredentialsInUse = append(keyCredentialsInUse, keyCredential)
		}
		if newestCredential.StartDateTime == nil || keyCredential.StartDateTime.After(*newestCredential.StartDateTime) {
			newestCredential = keyCredential
		}
	}
	return append(keyCredentialsInUse, newestCredential), nil
}

func (k keyCredential) new(resource v1.AzureAdApplication) (*msgraph.KeyCredential, *crypto.JwkPair, error) {
	jwkPair, err := crypto.GenerateJwkPair(resource)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate JWK pair for application: %w", err)
	}
	newKeyCredential := k.toKeyCredential(jwkPair)
	return &newKeyCredential, &jwkPair, nil
}

func (k keyCredential) toKeyCredential(jwkPair crypto.JwkPair) msgraph.KeyCredential {
	keyId := msgraph.UUID(uuid.New().String())
	keyBase64 := msgraph.Binary(jwkPair.PublicPem)
	return msgraph.KeyCredential{
		KeyID:       &keyId,
		DisplayName: ptr.String(util.DisplayName(time.Now())),
		Type:        ptr.String("AsymmetricX509Cert"),
		Usage:       ptr.String("Verify"),
		Key:         &keyBase64,
	}
}

func keyCredentialInUse(key msgraph.KeyCredential, keyIdsInUse []string) bool {
	for _, id := range keyIdsInUse {
		if string(*key.KeyID) == id {
			return true
		}
	}
	return false
}
