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
	return c.updateApplication(ctx, credential)
}

// TODO - revoke old keys, update other application metadata
func (c client) updateApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	clientId := credential.Status.ClientId
	objectId := credential.Status.ObjectId

	jwkPair, err := crypto.GenerateJwkPair(credential)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to generate JWK pair for application: %w", err)
	}

	// FIXME
	// keyCredential, err := c.addKeyCredential(ctx, credential, jwkPair)
	// if err != nil {
	// 	return azure.Application{}, fmt.Errorf("failed to update key credentials for application: %w", err)
	// }
	keyCredential := util.CreateKeyCredential(jwkPair)

	passwordCredential, err := c.addPasswordCredential(ctx, objectId)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to update password credentials for application: %w", err)
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
		PasswordKeyId:    string(*passwordCredential.KeyID),
		CertificateKeyId: string(*keyCredential.KeyID),
	}, nil
}

func (c client) addPasswordCredential(ctx context.Context, objectId string) (*msgraph.PasswordCredential, error) {
	requestParameter := addPasswordRequest()
	request := c.graphClient.Applications().ID(objectId).AddPassword(requestParameter).Request()
	response, err := request.Post(ctx)
	if err != nil {
		return &msgraph.PasswordCredential{}, err
	}
	return response, nil
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

func (c client) addKeyCredential(ctx context.Context, credential v1alpha1.AzureAdCredential, jwkPair crypto.JwkPair) (msgraph.KeyCredential, error) {
	requestParameter, err := addKeyRequest(credential, jwkPair)
	if err != nil {
		return msgraph.KeyCredential{}, err
	}

	request := c.graphClient.Applications().ID(credential.Status.ObjectId).AddKey(requestParameter).Request()
	keyCredentialResult, err := request.Post(ctx)
	if err != nil {
		return msgraph.KeyCredential{}, err
	}

	return *keyCredentialResult, nil
}

// FIXME: graph returns 401 unauthorized - should use existing/previous key to sign
func addKeyRequest(credential v1alpha1.AzureAdCredential, jwkPair crypto.JwkPair) (*msgraph.ApplicationAddKeyRequestParameter, error) {
	keyCredential := util.CreateKeyCredential(jwkPair)
	jwt, err := crypto.CreateSignedJwt(credential, jwkPair)
	if err != nil {
		return &msgraph.ApplicationAddKeyRequestParameter{}, err
	}
	return &msgraph.ApplicationAddKeyRequestParameter{
		KeyCredential: &keyCredential,
		Proof:         ptr.String(jwt),
	}, nil
}
