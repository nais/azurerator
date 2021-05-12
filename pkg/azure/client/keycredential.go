package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/util/crypto"
	strings2 "github.com/nais/azureator/pkg/util/strings"
)

type keyCredential struct {
	client
}

type addedKeyCredentialSet struct {
	Current addedKeyCredential
	Next    addedKeyCredential
}

type addedKeyCredential struct {
	KeyCredential msgraph.KeyCredential
	Jwk           crypto.Jwk
}

func (c client) keyCredential() keyCredential {
	return keyCredential{c}
}

// Generates a new set of key credentials, removing any key not in use (as indicated by AzureAdApplication.Status.CertificateKeyIds).
// With the exception of new applications, there should always be two active keys available at any given time so that running applications are not interfered with.
func (k keyCredential) rotate(tx azure.Transaction, existing azure.CredentialsSet, keyIdsInUse azure.KeyIdsInUse) (*msgraph.KeyCredential, *crypto.Jwk, error) {
	keyCredentialIdsInUse := append(
		keyIdsInUse.Certificate,
		existing.Current.Certificate.KeyId,
		existing.Next.Certificate.KeyId,
	)

	keysInUse, err := k.mapToKeyCredentials(tx, keyCredentialIdsInUse)
	if err != nil {
		return nil, nil, err
	}

	keyCredential, jwk, err := k.new(tx.Instance)
	if err != nil {
		return nil, nil, err
	}

	keysInUse = append(keysInUse, *keyCredential)

	app := util.EmptyApplication().Keys(keysInUse).Build()
	if err := k.application().patch(tx.Ctx, tx.Instance.GetObjectId(), app); err != nil {
		return nil, nil, fmt.Errorf("updating application with keycredential: %w", err)
	}

	return keyCredential, jwk, nil
}

func (k keyCredential) add(tx azure.Transaction) (*addedKeyCredentialSet, error) {
	application, err := k.Get(tx)
	if err != nil {
		return nil, err
	}

	currentKeyCredential, currentJwk, err := k.new(tx.Instance)
	if err != nil {
		return nil, err
	}

	nextKeyCredential, nextJwk, err := k.new(tx.Instance)
	if err != nil {
		return nil, err
	}

	application.KeyCredentials = append(application.KeyCredentials, *currentKeyCredential, *nextKeyCredential)

	app := util.EmptyApplication().Keys(application.KeyCredentials).Build()
	if err := k.application().patch(tx.Ctx, tx.Instance.GetObjectId(), app); err != nil {
		return nil, fmt.Errorf("updating application with keycredential set: %w", err)
	}

	return &addedKeyCredentialSet{
		Current: addedKeyCredential{
			KeyCredential: *currentKeyCredential,
			Jwk:           *currentJwk,
		},
		Next: addedKeyCredential{
			KeyCredential: *nextKeyCredential,
			Jwk:           *nextJwk,
		},
	}, nil
}

// Maps a list of key IDs to a list of KeyCredentials
func (k keyCredential) mapToKeyCredentials(tx azure.Transaction, keyIdsInUse []string) ([]msgraph.KeyCredential, error) {
	keyIdsInUse = strings2.RemoveDuplicates(keyIdsInUse)

	application, err := k.Get(tx)
	if err != nil {
		return nil, err
	}

	// Keep the newest registered credential in case the app already exists in Azure and is not referenced by resources in the cluster.
	// This case assumes the possibility of the Azure application being used in applications external to the cluster.
	// There should always be at least one keycredential registered for an application.
	var newestCredential msgraph.KeyCredential
	var keyCreatedByAzureratorFound = false
	for _, keyCredential := range application.KeyCredentials {
		keyDisplayName := *keyCredential.DisplayName
		if strings.HasPrefix(keyDisplayName, azure.AzureratorPrefix) {
			keyCreatedByAzureratorFound = true
		}
	}

	// Return early to prevent revoking keys for a pre-existing application that has been managed outside of azurerator
	if !keyCreatedByAzureratorFound {
		return application.KeyCredentials, nil
	}

	keyCredentialsInUse := make([]msgraph.KeyCredential, 0)
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

func (k keyCredential) new(resource v1.AzureAdApplication) (*msgraph.KeyCredential, *crypto.Jwk, error) {
	jwkPair, err := crypto.GenerateJwk(resource)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate JWK pair for application: %w", err)
	}

	newKeyCredential := k.toKeyCredential(jwkPair)

	return &newKeyCredential, &jwkPair, nil
}

func (k keyCredential) toKeyCredential(jwkPair crypto.Jwk) msgraph.KeyCredential {
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

func (k keyCredential) validate(tx azure.Transaction, existing azure.CredentialsSet) (bool, error) {
	app, err := k.Get(tx)
	if err != nil {
		return false, err
	}

	currentIsValid := false
	nextIsValid := false
	for _, credentials := range app.KeyCredentials {
		if string(*credentials.KeyID) == existing.Current.Certificate.KeyId {
			currentIsValid = true
		}
		if string(*credentials.KeyID) == existing.Next.Certificate.KeyId {
			nextIsValid = true
		}
	}

	return currentIsValid && nextIsValid, nil
}

func keyCredentialInUse(key msgraph.KeyCredential, keyIdsInUse []string) bool {
	keyId := string(*key.KeyID)
	for _, id := range keyIdsInUse {
		if keyId == id {
			return true
		}
	}
	return false
}
