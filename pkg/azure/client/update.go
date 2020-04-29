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

type NewKeyCredentials struct {
	KeyCredential msgraph.KeyCredential
	JwkPair       crypto.JwkPair
}

// Update updates an existing AAD application
func (c client) Update(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	return c.updateApplication(ctx, credential)
}

// TODO - revoke old keys, update other application metadata
func (c client) updateApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	clientId := credential.Status.ClientId
	objectId := credential.Status.ObjectId

	passwordCredential, err := c.addPasswordCredential(ctx, objectId)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to update password credentials for application: %w", err)
	}

	newKeyCredentials, err := c.addKeyCredential(ctx, credential)
	if err != nil {
		return azure.Application{}, fmt.Errorf("failed to update key credentials for application: %w", err)
	}

	return azure.Application{
		Credentials: azure.Credentials{
			Public: azure.Public{
				ClientId: clientId,
				Jwk:      newKeyCredentials.JwkPair.Public,
			},
			Private: azure.Private{
				ClientId:     clientId,
				ClientSecret: *passwordCredential.SecretText,
				Jwk:          newKeyCredentials.JwkPair.Private,
			},
		},
		ClientId:         clientId,
		ObjectId:         objectId,
		CertificateKeyId: string(*newKeyCredentials.KeyCredential.KeyID),
		PasswordKeyId:    string(*passwordCredential.KeyID),
	}, nil
}

func (c client) addPasswordCredential(ctx context.Context, objectId string) (msgraph.PasswordCredential, error) {
	requestParameter := addPasswordRequest()
	request := c.graphClient.Applications().ID(objectId).AddPassword(requestParameter).Request()
	response, err := request.Post(ctx)
	if err != nil {
		return msgraph.PasswordCredential{}, err
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

func (c client) addKeyCredential(ctx context.Context, credential v1alpha1.AzureAdCredential) (NewKeyCredentials, error) {
	existingKeyCredential, err := c.getExistingKeyCredential(ctx, credential)
	jwkPair, err := crypto.GenerateJwkPair(credential)
	if err != nil {
		return NewKeyCredentials{}, fmt.Errorf("failed to generate JWK pair for application: %w", err)
	}
	newKeyCredential := util.ToKeyCredential(jwkPair)
	if err := c.graphClient.Applications().ID(credential.Status.ObjectId).Request().Update(ctx, &msgraph.Application{
		KeyCredentials: []msgraph.KeyCredential{
			newKeyCredential,
			existingKeyCredential,
		},
	}); err != nil {
		return NewKeyCredentials{}, fmt.Errorf("failed to update application with keycredential: %w", err)
	}
	return NewKeyCredentials{
		KeyCredential: newKeyCredential,
		JwkPair:       jwkPair,
	}, nil
}
