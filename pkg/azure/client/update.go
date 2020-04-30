package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Update updates an existing AAD application
func (c client) Update(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	clientId := credential.Status.ClientId
	objectId := credential.Status.ObjectId

	passwordCredential, err := c.addPasswordCredential(ctx, objectId)
	if err != nil {
		return azure.Application{}, err
	}
	keyCredential, jwkPair, err := c.rotateKeyCredential(ctx, credential)
	if err != nil {
		return azure.Application{}, err
	}

	app := updateApplicationTemplate(credential)
	if err := c.updateApplication(ctx, objectId, app); err != nil {
		return azure.Application{}, err
	}

	return azure.Application{
		Credentials: azure.Credentials{
			Public: azure.Public{
				ClientId: clientId,
				Jwk:      jwkPair.Public,
			},
			Private: azure.Private{
				ClientId:     clientId,
				ClientSecret: *passwordCredential.SecretText,
				Jwk:          jwkPair.Private,
			},
		},
		ClientId:         clientId,
		ObjectId:         objectId,
		CertificateKeyId: string(*keyCredential.KeyID),
		PasswordKeyId:    string(*passwordCredential.KeyID),
	}, nil
}

func (c client) updateApplication(ctx context.Context, id string, application *msgraph.Application) error {
	if err := c.graphClient.Applications().ID(id).Request().Update(ctx, application); err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}
	return nil
}

// TODO - revoke expired keys not in use
func (c client) addPasswordCredential(ctx context.Context, objectId string) (msgraph.PasswordCredential, error) {
	requestParameter := addPasswordRequest()
	request := c.graphClient.Applications().ID(objectId).AddPassword(requestParameter).Request()
	response, err := request.Post(ctx)
	if err != nil {
		return msgraph.PasswordCredential{}, fmt.Errorf("failed to add password credentials for application: %w", err)
	}
	return *response, nil
}

// TODO - validity, unique displayname
func addPasswordRequest() *msgraph.ApplicationAddPasswordRequestParameter {
	startDateTime := time.Now()
	endDateTime := time.Now().AddDate(1, 0, 0)
	keyId := msgraph.UUID(uuid.New().String())
	return &msgraph.ApplicationAddPasswordRequestParameter{
		PasswordCredential: &msgraph.PasswordCredential{
			StartDateTime: &startDateTime,
			EndDateTime:   &endDateTime,
			KeyID:         &keyId,
			DisplayName:   ptr.String("azurerator"),
		},
	}
}

// Generates a new set of key credentials, removing any key not in use (as indicated by AzureAdCredential.Status.CertificateKeyId).
// There should always be two active keys available at any given time so that running applications are not interfered with.
func (c client) rotateKeyCredential(ctx context.Context, credential v1alpha1.AzureAdCredential) (msgraph.KeyCredential, crypto.JwkPair, error) {
	existingKeyCredential, err := c.getExistingKeyCredential(ctx, credential)
	keyCredential, jwkPair, err := util.GenerateNewKeyCredentialFor(credential)
	if err != nil {
		return msgraph.KeyCredential{}, crypto.JwkPair{}, err
	}
	keys := []msgraph.KeyCredential{keyCredential, existingKeyCredential}
	app := util.EmptyApplication().Keys(keys).Build()
	if err := c.updateApplication(ctx, credential.Status.ObjectId, app); err != nil {
		return msgraph.KeyCredential{}, crypto.JwkPair{}, fmt.Errorf("failed to update application with keycredential: %w", err)
	}
	return keyCredential, jwkPair, nil
}

func (c client) setApplicationIdentifierUri(ctx context.Context, application msgraph.Application) error {
	identifierUri := util.IdentifierUri(*application.AppID)
	app := util.EmptyApplication().IdentifierUri(identifierUri).Build()
	if err := c.updateApplication(ctx, *application.ID, app); err != nil {
		return fmt.Errorf("failed to add application identifier URI: %w", err)
	}
	return nil
}

// TODO - update other application metadata, preauthorizedapps
func updateApplicationTemplate(credential v1alpha1.AzureAdCredential) *msgraph.Application {
	uri := util.IdentifierUri(credential.Status.ClientId)
	return util.EmptyApplication().IdentifierUri(uri).Build()
}
