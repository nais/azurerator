package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type passwordCredential struct {
	client
}

func (c client) passwordCredential() passwordCredential {
	return passwordCredential{c}
}

func (p passwordCredential) rotate(tx azure.Transaction) (msgraph.PasswordCredential, error) {
	app, err := p.Get(tx)
	if err != nil {
		return msgraph.PasswordCredential{}, err
	}
	newCred, err := p.add(tx.Ctx, *app.ID)
	if err != nil {
		return msgraph.PasswordCredential{}, err
	}
	for _, cred := range app.PasswordCredentials {
		keyId := string(*cred.KeyID)
		isNewCredKeyId := keyId == string(*newCred.KeyID)
		isPreviousKeyId := keyId == tx.Instance.Status.PasswordKeyId
		if isPreviousKeyId || isNewCredKeyId {
			continue
		}
		if err := p.remove(tx.Ctx, *app.ID, cred.KeyID); err != nil {
			return msgraph.PasswordCredential{}, err
		}
	}
	return newCred, nil
}

func (p passwordCredential) add(ctx context.Context, id azure.ObjectId) (msgraph.PasswordCredential, error) {
	requestParameter := p.toAddRequest()
	request := p.graphClient.Applications().ID(id).AddPassword(requestParameter).Request()
	response, err := request.Post(ctx)
	if err != nil {
		return msgraph.PasswordCredential{}, fmt.Errorf("failed to add password credentials for application: %w", err)
	}
	return *response, nil
}

func (p passwordCredential) remove(ctx context.Context, id azure.ClientId, keyId *msgraph.UUID) error {
	req := p.toRemoveRequest(keyId)
	if err := p.graphClient.Applications().ID(id).RemovePassword(req).Request().Post(ctx); err != nil {
		return fmt.Errorf("failed to remove password credential: %w", err)
	}
	return nil
}

func (p passwordCredential) toAddRequest() *msgraph.ApplicationAddPasswordRequestParameter {
	startDateTime := time.Now()
	endDateTime := time.Now().AddDate(1, 0, 0)
	keyId := msgraph.UUID(uuid.New().String())
	return &msgraph.ApplicationAddPasswordRequestParameter{
		PasswordCredential: &msgraph.PasswordCredential{
			StartDateTime: &startDateTime,
			EndDateTime:   &endDateTime,
			KeyID:         &keyId,
			DisplayName:   ptr.String(util.DisplayName()),
		},
	}
}

func (p passwordCredential) toRemoveRequest(keyId *msgraph.UUID) *msgraph.ApplicationRemovePasswordRequestParameter {
	return &msgraph.ApplicationRemovePasswordRequestParameter{
		KeyID: keyId,
	}
}
