package passwordcredential

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application"
	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/transaction"
	"github.com/nais/azureator/pkg/azure/util"
	stringutils "github.com/nais/azureator/pkg/util/strings"
)

type PasswordCredential interface {
	Add(tx transaction.Transaction) (msgraph.PasswordCredential, error)
	DeleteExpired(tx transaction.Transaction) error
	DeleteUnused(tx transaction.Transaction) error
	Purge(tx transaction.Transaction) error
	Rotate(tx transaction.Transaction) (*msgraph.PasswordCredential, error)
	Validate(tx transaction.Transaction, existing credentials.Set) (bool, error)
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

	response, err := request.Post(tx.Ctx)
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
				tx.Log.Debugf("revoking expired password credential '%s' (ID: %s, expired: %s)", *cred.DisplayName, *cred.KeyID, cred.EndDateTime)
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
			tx.Log.Debugf("revoking unused password credential '%s' (ID: %s)", *cred.DisplayName, *cred.KeyID)
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

	// sleep to prevent concurrent modification error from Microsoft
	time.Sleep(p.DelayIntervalBetweenModifications())

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

func (p passwordCredential) Validate(tx transaction.Transaction, existing credentials.Set) (bool, error) {
	app, err := p.Application().Get(tx)
	if err != nil {
		return false, err
	}

	currentIsValid := false
	nextIsValid := false
	for _, cred := range app.PasswordCredentials {
		notExpired := cred.EndDateTime.After(time.Now())

		currentIdMatches := string(*cred.KeyID) == existing.Current.Password.KeyId
		if currentIdMatches && notExpired {
			currentIsValid = true
		}

		nextIdMatches := string(*cred.KeyID) == existing.Next.Password.KeyId
		if nextIdMatches && notExpired {
			nextIsValid = true
		}
	}

	return currentIsValid && nextIsValid, nil
}

func (p passwordCredential) remove(tx transaction.Transaction, id azure.ClientId, keyId *msgraph.UUID) error {
	req := p.toRemoveRequest(keyId)
	if err := p.GraphClient().Applications().ID(id).RemovePassword(req).Request().Post(tx.Ctx); err != nil {
		// Microsoft returns HTTP 500 sometimes after adding new credentials due to concurrent modifications; we'll ignore this on our end for now
		tx.Log.Errorf("removing password credential with id '%s': '%v'; ignoring", string(*keyId), err)
	}
	return nil
}

func (p passwordCredential) toAddRequest(tx transaction.Transaction) *msgraph.ApplicationAddPasswordRequestParameter {
	startDateTime := time.Now()

	var endDateTime time.Time
	if tx.Instance.Spec.SecretProtected {
		endDateTime = startDateTime.AddDate(99, 0, 0)
	} else {
		endDateTime = startDateTime.AddDate(1, 0, 0)
	}

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

func (p passwordCredential) revocationCandidates(tx transaction.Transaction, app msgraph.Application) []msgraph.PasswordCredential {
	keyIdsInUse := append(
		tx.Secrets.KeyIDs.Used.Password,
		tx.Secrets.LatestCredentials.Set.Current.Password.KeyId,
		tx.Secrets.LatestCredentials.Set.Next.Password.KeyId,
	)
	keyIdsInUse = stringutils.RemoveDuplicates(keyIdsInUse)

	// Keep the newest registered credential in case the app already exists in Azure and is not referenced by resources in the cluster.
	// This case assumes the possibility of the Azure application being used in applications external to the cluster.
	// There should always be at least one passwordcredential registered for an application.
	var newestCredential msgraph.PasswordCredential
	var newestCredentialIndex int
	var keyCreatedByAzureratorFound = false

	for i, passwordCredential := range app.PasswordCredentials {
		if newestCredential.StartDateTime == nil || passwordCredential.StartDateTime.After(*newestCredential.StartDateTime) {
			newestCredential = passwordCredential
			newestCredentialIndex = i
		}

		if passwordCredential.DisplayName == nil {
			continue
		}

		keyDisplayName := *passwordCredential.DisplayName
		if strings.HasPrefix(keyDisplayName, azure.AzureratorPrefix) {
			keyCreatedByAzureratorFound = true
		}
	}

	// Return early to prevent revoking keys for a pre-existing application that has been managed outside of azurerator
	if !keyCreatedByAzureratorFound {
		return make([]msgraph.PasswordCredential, 0)
	}

	revoked := make([]msgraph.PasswordCredential, 0)
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
