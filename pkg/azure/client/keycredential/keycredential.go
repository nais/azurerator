package keycredential

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application"
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/transaction"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/util/crypto"
	stringutils "github.com/nais/azureator/pkg/util/strings"
)

type KeyCredential interface {
	Add(tx transaction.Transaction) (*credentials.AddedKeyCredentialSet, error)
	DeleteExpired(tx transaction.Transaction) error
	DeleteUnused(tx transaction.Transaction) error
	Purge(tx transaction.Transaction) error
	Rotate(tx transaction.Transaction) (*msgraph.KeyCredential, *crypto.Jwk, error)
	Validate(tx transaction.Transaction, existing credentials.Set) (bool, error)
}

type keyCredential struct {
	Client
}

type Client interface {
	Application() application.Application
}

// Workaround to include empty array of KeyCredentials in JSON serialization.
// The autogenerated library code uses 'omitempty' for KeyCredentials, which when empty
// leaves the list of redirect URIs unchanged and non-empty.
type app struct {
	msgraph.DirectoryObject
	KeyCredentials []msgraph.KeyCredential `json:"keyCredentials"`
}

func NewKeyCredential(client Client) KeyCredential {
	return keyCredential{Client: client}
}

func (k keyCredential) Add(tx transaction.Transaction) (*credentials.AddedKeyCredentialSet, error) {
	actualApp, err := k.Client.Application().Get(tx)
	if err != nil {
		return nil, err
	}

	currentKeyCredential, currentJwk, err := k.new(tx)
	if err != nil {
		return nil, err
	}

	nextKeyCredential, nextJwk, err := k.new(tx)
	if err != nil {
		return nil, err
	}

	actualApp.KeyCredentials = append(actualApp.KeyCredentials, *currentKeyCredential, *nextKeyCredential)

	app := util.EmptyApplication().Keys(actualApp.KeyCredentials).Build()
	if err := k.Application().Patch(tx.Ctx, tx.Instance.GetObjectId(), app); err != nil {
		return nil, fmt.Errorf("updating application with keycredential set: %w", err)
	}

	return &credentials.AddedKeyCredentialSet{
		Current: credentials.AddedKeyCredential{
			KeyCredential: *currentKeyCredential,
			Jwk:           *currentJwk,
		},
		Next: credentials.AddedKeyCredential{
			KeyCredential: *nextKeyCredential,
			Jwk:           *nextJwk,
		},
	}, nil
}

func (k keyCredential) DeleteExpired(tx transaction.Transaction) error {
	actualApp, err := k.Application().Get(tx)
	if err != nil {
		return err
	}

	desiredCredentials := make([]msgraph.KeyCredential, 0)

	for _, cred := range actualApp.KeyCredentials {
		notExpired := cred.EndDateTime.After(time.Now())
		if notExpired {
			desiredCredentials = append(desiredCredentials, cred)
		} else if cred.DisplayName != nil && cred.KeyID != nil {
			tx.Log.Debugf("revoking expired key credential '%s' (ID: %s, expired: %s)", *cred.DisplayName, *cred.KeyID, cred.EndDateTime)
		}
	}

	app := &app{
		KeyCredentials: desiredCredentials,
	}
	return k.Application().Patch(tx.Ctx, tx.Instance.GetObjectId(), app)
}

func (k keyCredential) DeleteUnused(tx transaction.Transaction) error {
	keysInUse, err := k.mapToKeyCredentials(tx)
	if err != nil {
		return err
	}

	app := util.EmptyApplication().Keys(keysInUse).Build()
	if err := k.Application().Patch(tx.Ctx, tx.Instance.GetObjectId(), app); err != nil {
		return fmt.Errorf("updating application with keycredential: %w", err)
	}

	return nil
}

// Rotate generates a new set of key credentials, removing any key not in use (as indicated by AzureAdApplication.Status.CertificateKeyIds).
// Except new applications, there should always be at least two active keys available at any given time so that running applications are not interfered with.
func (k keyCredential) Rotate(tx transaction.Transaction) (*msgraph.KeyCredential, *crypto.Jwk, error) {
	keysInUse, err := k.mapToKeyCredentials(tx)
	if err != nil {
		return nil, nil, err
	}

	keyCredential, jwk, err := k.new(tx)
	if err != nil {
		return nil, nil, err
	}

	keysInUse = append(keysInUse, *keyCredential)

	app := util.EmptyApplication().Keys(keysInUse).Build()
	if err := k.Application().Patch(tx.Ctx, tx.Instance.GetObjectId(), app); err != nil {
		return nil, nil, fmt.Errorf("updating application with keycredential: %w", err)
	}

	return keyCredential, jwk, nil
}

func (k keyCredential) Purge(tx transaction.Transaction) error {
	app := &app{
		KeyCredentials: make([]msgraph.KeyCredential, 0),
	}

	return k.Application().Patch(tx.Ctx, tx.Instance.GetObjectId(), app)
}

func (k keyCredential) Validate(tx transaction.Transaction, existing credentials.Set) (bool, error) {
	app, err := k.Application().Get(tx)
	if err != nil {
		return false, err
	}

	currentIsValid := false
	nextIsValid := false
	for _, cred := range app.KeyCredentials {
		notExpired := cred.EndDateTime.After(time.Now())

		currentIdMatches := string(*cred.KeyID) == existing.Current.Certificate.KeyId
		if currentIdMatches && notExpired {
			currentIsValid = true
		}

		nextIdMatches := string(*cred.KeyID) == existing.Next.Certificate.KeyId
		if nextIdMatches && notExpired {
			nextIsValid = true
		}
	}

	return currentIsValid && nextIsValid, nil
}

// Maps a list of key IDs to a list of KeyCredentials
func (k keyCredential) mapToKeyCredentials(tx transaction.Transaction) ([]msgraph.KeyCredential, error) {
	keyIdsInUse := append(
		tx.Secrets.KeyIDs.Used.Certificate,
		tx.Secrets.LatestCredentials.Set.Current.Certificate.KeyId,
		tx.Secrets.LatestCredentials.Set.Next.Certificate.KeyId,
	)
	keyIdsInUse = stringutils.RemoveDuplicates(keyIdsInUse)

	actualApp, err := k.Application().Get(tx)
	if err != nil {
		return nil, err
	}

	// Keep the newest registered credential in case the app already exists in Azure and is not referenced by resources in the cluster.
	// This case assumes the possibility of the Azure application being used in applications external to the cluster.
	// There should always be at least one keycredential registered for an application.
	var newestCredential msgraph.KeyCredential
	var keyCreatedByAzureratorFound = false
	for _, keyCredential := range actualApp.KeyCredentials {
		keyDisplayName := *keyCredential.DisplayName
		if strings.HasPrefix(keyDisplayName, azure.AzureratorPrefix) {
			keyCreatedByAzureratorFound = true
		}
	}

	// Return early to prevent revoking keys for a pre-existing application that has been managed outside of azurerator
	if !keyCreatedByAzureratorFound {
		return actualApp.KeyCredentials, nil
	}

	keyCredentialsInUse := make([]msgraph.KeyCredential, 0)
	for _, keyCredential := range actualApp.KeyCredentials {
		if keyCredentialInUse(keyCredential, keyIdsInUse) {
			keyCredentialsInUse = append(keyCredentialsInUse, keyCredential)
		} else if keyCredential.DisplayName != nil && keyCredential.KeyID != nil {
			tx.Log.Debugf("revoking unused key credential '%s' (ID: %s)", *keyCredential.DisplayName, *keyCredential.KeyID)
		}

		if newestCredential.StartDateTime == nil || keyCredential.StartDateTime.After(*newestCredential.StartDateTime) {
			newestCredential = keyCredential
		}
	}
	return append(keyCredentialsInUse, newestCredential), nil
}

func (k keyCredential) new(tx transaction.Transaction) (*msgraph.KeyCredential, *crypto.Jwk, error) {
	jwkPair, err := crypto.GenerateJwk(tx.Instance, tx.ClusterName)
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

func keyCredentialInUse(key msgraph.KeyCredential, keyIdsInUse []string) bool {
	keyId := string(*key.KeyID)
	for _, id := range keyIdsInUse {
		if keyId == id {
			return true
		}
	}
	return false
}
