package client

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"gopkg.in/square/go-jose.v2"
)

// Update updates an existing AAD application
func (c client) Update(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	return c.updateApplication(ctx, credential)
}

// TODO
func (c client) updateApplication(ctx context.Context, credential v1alpha1.AzureAdCredential) (azure.Application, error) {
	return azure.Application{
		Credentials: azure.Credentials{
			Public: azure.Public{
				ClientId: credential.Status.ClientId,
				Jwk:      jose.JSONWebKey{},
			},
			Private: azure.Private{
				ClientId:     credential.Status.ClientId,
				ClientSecret: "",
				Jwk:          jose.JSONWebKey{},
			},
		},
		ClientId:         credential.Status.ClientId,
		ObjectId:         credential.Status.ObjectId,
		PasswordKeyId:    "",
		CertificateKeyId: "",
	}, nil
}

// TODO - validity
func (c client) addPasswordCredential(ctx context.Context, objectId string) (*msgraph.PasswordCredential, error) {
	startDateTime := time.Now()
	endDateTime := time.Now().AddDate(1, 0, 0)
	keyId := msgraph.UUID(uuid.New().String())
	password := &msgraph.ApplicationAddPasswordRequestParameter{
		PasswordCredential: &msgraph.PasswordCredential{
			StartDateTime: &startDateTime,
			EndDateTime:   &endDateTime,
			KeyID:         &keyId,
			DisplayName:   ptr.String("azurerator"),
		},
	}
	request := c.graphClient.Applications().ID(objectId).AddPassword(password).Request()
	response, err := request.Post(ctx)
	if err != nil {
		return &msgraph.PasswordCredential{}, err
	}
	return response, nil
}
