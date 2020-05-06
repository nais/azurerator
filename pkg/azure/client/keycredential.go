package client

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/util/crypto"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Generates a new set of key credentials, removing any key not in use (as indicated by AzureAdApplication.Status.CertificateKeyId).
// There should always be two active keys available at any given time so that running applications are not interfered with.
func (c client) rotateKeyCredential(tx azure.Transaction) (msgraph.KeyCredential, crypto.JwkPair, error) {
	existingKeyCredential, err := c.getExistingKeyCredential(tx)
	keyCredential, jwkPair, err := util.GenerateNewKeyCredentialFor(tx.Resource)
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
	for _, keyCredential := range application.KeyCredentials {
		if string(*keyCredential.KeyID) == tx.Resource.Status.CertificateKeyId {
			return keyCredential, nil
		}
	}
	return msgraph.KeyCredential{}, fmt.Errorf("failed to find application key matching the previous key ID in Status field")
}
