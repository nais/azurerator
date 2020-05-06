package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

func (c client) rotatePasswordCredential(ctx context.Context, resource v1alpha1.AzureAdApplication) (msgraph.PasswordCredential, error) {
	app, err := c.Get(ctx, resource)
	if err != nil {
		return msgraph.PasswordCredential{}, err
	}
	newCred, err := c.addPasswordCredential(ctx, *app.ID)
	if err != nil {
		return msgraph.PasswordCredential{}, err
	}
	for _, cred := range app.PasswordCredentials {
		keyId := string(*cred.KeyID)
		isNewCredKeyId := keyId == string(*newCred.KeyID)
		isPreviousKeyId := keyId == resource.Status.PasswordKeyId
		if isPreviousKeyId || isNewCredKeyId {
			continue
		}
		if err := c.removePasswordCredential(ctx, *app.ID, cred.KeyID); err != nil {
			return msgraph.PasswordCredential{}, err
		}
	}
	return newCred, nil
}

func (c client) addPasswordCredential(ctx context.Context, objectId string) (msgraph.PasswordCredential, error) {
	requestParameter := addPasswordRequest()
	request := c.graphClient.Applications().ID(objectId).AddPassword(requestParameter).Request()
	response, err := request.Post(ctx)
	if err != nil {
		return msgraph.PasswordCredential{}, fmt.Errorf("failed to add password credentials for application: %w", err)
	}
	return *response, nil
}

func (c client) removePasswordCredential(ctx context.Context, appId string, keyId *msgraph.UUID) error {
	req := removePasswordRequest(keyId)
	if err := c.graphClient.Applications().ID(appId).RemovePassword(req).Request().Post(ctx); err != nil {
		return fmt.Errorf("failed to remove password credential: %w", err)
	}
	return nil
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

func removePasswordRequest(keyId *msgraph.UUID) *msgraph.ApplicationRemovePasswordRequestParameter {
	return &msgraph.ApplicationRemovePasswordRequestParameter{
		KeyID: keyId,
	}
}
