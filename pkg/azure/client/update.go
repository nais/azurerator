package client

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
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

	passwordCredential, err := c.addPasswordCredential(ctx, objectId)
	if err != nil {
		return azure.Application{}, err
	}

	return azure.Application{
		Credentials: azure.Credentials{
			Public: azure.Public{
				ClientId: clientId,
			},
			Private: azure.Private{
				ClientId:     clientId,
				ClientSecret: *passwordCredential.SecretText,
			},
		},
		ClientId:      clientId,
		ObjectId:      objectId,
		PasswordKeyId: string(*passwordCredential.KeyID),
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
