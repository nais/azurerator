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

func (p passwordCredential) rotate(tx azure.Transaction, keyIdsInUse []string) (msgraph.PasswordCredential, error) {
	app, err := p.Get(tx)
	if err != nil {
		return msgraph.PasswordCredential{}, err
	}
	newCred, err := p.add(tx.Ctx, *app.ID)
	if err != nil {
		return msgraph.PasswordCredential{}, err
	}
	revocationCandidates := p.revocationCandidates(app, append(keyIdsInUse, string(*newCred.KeyID)))
	for _, cred := range revocationCandidates {
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
			DisplayName:   ptr.String(util.DisplayName(time.Now())),
		},
	}
}

func (p passwordCredential) toRemoveRequest(keyId *msgraph.UUID) *msgraph.ApplicationRemovePasswordRequestParameter {
	return &msgraph.ApplicationRemovePasswordRequestParameter{
		KeyID: keyId,
	}
}

func (p passwordCredential) revocationCandidates(app msgraph.Application, keyIdsInUse []string) []msgraph.PasswordCredential {
	revoked := make([]msgraph.PasswordCredential, 0)

	// Keep the newest registered credential in case the app already exists in Azure and is not referenced by resources in the cluster.
	// This case assumes the possibility of the Azure application being used in applications external to the cluster.
	// There should always be at least one passwordcredential registered for an application.
	var newestCredential msgraph.PasswordCredential
	var newestCredentialIndex int
	for i, passwordCredential := range app.PasswordCredentials {
		if newestCredential.StartDateTime == nil || passwordCredential.StartDateTime.After(*newestCredential.StartDateTime) {
			newestCredential = passwordCredential
			newestCredentialIndex = i
		}
	}
	for i, passwordCredential := range app.PasswordCredentials {
		if isPasswordInUse(passwordCredential, keyIdsInUse) || i == newestCredentialIndex {
			continue
		}
		revoked = append(revoked, passwordCredential)
	}
	return revoked
}

func isPasswordInUse(cred msgraph.PasswordCredential, idsInUse []string) bool {
	keyId := string(*cred.KeyID)
	for _, idInUse := range idsInUse {
		if keyId == idInUse {
			return true
		}
	}
	return false
}
