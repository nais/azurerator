package passwordcredential

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application"
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/transaction"
)

type PasswordCredential interface {
	Add(tx transaction.Transaction) (msgraph.PasswordCredential, error)
	DeleteExpired(tx transaction.Transaction) error
	DeleteUnused(tx transaction.Transaction) error
	Purge(tx transaction.Transaction) error
	Rotate(tx transaction.Transaction) (*msgraph.PasswordCredential, error)
}

type passwordCredential struct {
	Client
}

type Client interface {
	azure.RuntimeClient
	Application() application.Application
}

func NewPasswordCredential(client Client) PasswordCredential {
	return passwordCredential{Client: client}
}

func (p passwordCredential) Add(tx transaction.Transaction) (msgraph.PasswordCredential, error) {
	objectId := tx.Instance.GetObjectId()

	requestParameter := p.toAddRequest(tx)

	request := p.GraphClient().Applications().ID(objectId).AddPassword(requestParameter).Request()

	var response *msgraph.PasswordCredential
	err := credentials.RetryGraph(tx.Ctx, &tx.Logger, "add password credential", func(ctx context.Context) error {
		var err error
		response, err = request.Post(ctx)
		return err
	})
	if err != nil {
		return msgraph.PasswordCredential{}, fmt.Errorf("adding password credentials for application: %w", err)
	}

	return *response, nil
}

func (p passwordCredential) DeleteExpired(tx transaction.Transaction) error {
	app, err := p.Application().Get(tx)
	if err != nil {
		return err
	}

	for _, cred := range app.PasswordCredentials {
		expired := cred.EndDateTime.Before(time.Now())

		if expired {
			if cred.DisplayName != nil && cred.KeyID != nil {
				tx.Logger.Debugf("revoking expired password credential '%s' (ID: %s, expired: %s)", *cred.DisplayName, *cred.KeyID, cred.EndDateTime)
			}

			if err := p.remove(tx, *app.ID, cred.KeyID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p passwordCredential) DeleteUnused(tx transaction.Transaction) error {
	app, err := p.Application().Get(tx)
	if err != nil {
		return err
	}

	revocationCandidates := p.revocationCandidates(tx, app)
	for _, cred := range revocationCandidates {
		if cred.DisplayName != nil && cred.KeyID != nil {
			tx.Logger.Debugf("revoking unused password credential '%s' (ID: %s)", *cred.DisplayName, *cred.KeyID)
		}

		if err := p.remove(tx, *app.ID, cred.KeyID); err != nil {
			return err
		}
	}

	return nil
}

func (p passwordCredential) Rotate(tx transaction.Transaction) (*msgraph.PasswordCredential, error) {
	app, err := p.Application().Get(tx)
	if err != nil {
		return nil, err
	}

	revocationCandidates := p.revocationCandidates(tx, app)
	for _, cred := range revocationCandidates {
		if err := p.remove(tx, *app.ID, cred.KeyID); err != nil {
			return nil, err
		}
	}

	if err := credentials.Wait(tx.Ctx, p.DelayIntervalBetweenModifications()); err != nil {
		return nil, err
	}

	newCred, err := p.Add(tx)
	if err != nil {
		return nil, err
	}

	return &newCred, nil
}

func (p passwordCredential) Purge(tx transaction.Transaction) error {
	app, err := p.Application().Get(tx)
	if err != nil {
		return err
	}

	for _, cred := range app.PasswordCredentials {
		if err := p.remove(tx, *app.ID, cred.KeyID); err != nil {
			return err
		}
	}

	return nil
}

func (p passwordCredential) remove(tx transaction.Transaction, id azure.ClientId, keyId *msgraph.UUID) error {
	if err := credentials.Wait(tx.Ctx, p.DelayIntervalBetweenModifications()); err != nil {
		return err
	}

	req := p.toRemoveRequest(keyId)
	if err := credentials.RetryGraph(tx.Ctx, &tx.Logger, "remove password credential", func(ctx context.Context) error {
		return p.GraphClient().Applications().ID(id).RemovePassword(req).Request().Post(ctx)
	}); err != nil {
		// a failed response can hide a successful removal
		app, getErr := p.Application().Get(tx)
		removed := getErr == nil && !slices.ContainsFunc(app.PasswordCredentials, func(c msgraph.PasswordCredential) bool {
			return c.KeyID != nil && *c.KeyID == *keyId
		})
		if removed {
			return nil
		}
		return fmt.Errorf("removing password credential: %w", err)
	}
	return nil
}

func (p passwordCredential) toAddRequest(tx transaction.Transaction) *msgraph.ApplicationAddPasswordRequestParameter {
	startDateTime := time.Now()

	endDateTime := startDateTime.AddDate(1, 0, 0)

	keyId := msgraph.UUID(uuid.New().String())

	return &msgraph.ApplicationAddPasswordRequestParameter{
		PasswordCredential: &msgraph.PasswordCredential{
			StartDateTime: &startDateTime,
			EndDateTime:   &endDateTime,
			KeyID:         &keyId,
			DisplayName:   new(util.DisplayName(time.Now())),
		},
	}
}

func (p passwordCredential) toRemoveRequest(keyId *msgraph.UUID) *msgraph.ApplicationRemovePasswordRequestParameter {
	return &msgraph.ApplicationRemovePasswordRequestParameter{
		KeyID: keyId,
	}
}

// revocationCandidates returns the passwords that the managed Secrets do not use.
func (p passwordCredential) revocationCandidates(tx transaction.Transaction, app msgraph.Application) []msgraph.PasswordCredential {
	inUse := append(
		slices.Clone(tx.Secrets.KeyIDs.Used.Password),
		tx.Secrets.LatestCredentials.Set.Current.Password.KeyId,
		tx.Secrets.LatestCredentials.Set.Next.Password.KeyId,
	)

	revoked := make([]msgraph.PasswordCredential, 0)
	for _, password := range app.PasswordCredentials {
		if password.KeyID != nil && slices.Contains(inUse, string(*password.KeyID)) {
			continue
		}
		revoked = append(revoked, password)
	}

	return revoked
}
