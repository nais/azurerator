package federatedcredential

import (
	"fmt"
	"time"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/transaction"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"
)

type FederatedCredential interface {
	Process(tx transaction.Transaction) error
}

type federatedCredential struct {
	azure.RuntimeClient
}

func NewFederatedCredential(client azure.RuntimeClient) FederatedCredential {
	return federatedCredential{RuntimeClient: client}
}

func (f federatedCredential) Process(tx transaction.Transaction) error {
	objectID := tx.Instance.GetObjectId()
	collection := f.GraphClient().Applications().ID(objectID).FederatedIdentityCredentials()
	logger := tx.Logger.WithField("subsystem", "federatedcredential")

	existing, err := collection.Request().GetN(tx.Ctx, f.MaxNumberOfPagesToFetch())
	if err != nil {
		return fmt.Errorf("listing federated identity credentials for application: %w", err)
	}

	changes, err := diff(existing, tx.Instance.Spec.FederatedCredentials)
	if err != nil {
		return err
	}

	// Deletions free unique issuer-subject pairs before updates and creations claim them.
	for _, credential := range changes.toDelete {
		time.Sleep(f.DelayIntervalBetweenModifications())
		if err := collection.ID(*credential.ID).Request().Delete(tx.Ctx); err != nil {
			return fmt.Errorf("deleting federated identity credential %q: %w", stringValue(credential.Name), err)
		}
		logger.Debugf("deleted federated identity credential %q", stringValue(credential.Name))
	}

	for _, update := range changes.toUpdate {
		time.Sleep(f.DelayIntervalBetweenModifications())
		credential := fromDesired(update.desired)
		// Names are immutable and must not be included in a Graph PATCH request.
		credential.Name = nil
		if err := collection.ID(update.id).Request().Update(tx.Ctx, &credential); err != nil {
			return fmt.Errorf("updating federated identity credential %q: %w", update.desired.Name, err)
		}
		logger.Debugf("updated federated identity credential %q", update.desired.Name)
	}

	for _, desired := range changes.toCreate {
		credential := fromDesired(desired)
		time.Sleep(f.DelayIntervalBetweenModifications())
		if _, err := collection.Request().Add(tx.Ctx, &credential); err != nil {
			return fmt.Errorf("creating federated identity credential %q: %w", desired.Name, err)
		}
		logger.Debugf("created federated identity credential %q", desired.Name)
	}

	return nil
}

type (
	changes struct {
		toCreate []v1.AzureAdFederatedCredential
		toUpdate []credentialUpdate
		toDelete []msgraph.FederatedIdentityCredential
	}
	credentialUpdate struct {
		id      string
		desired v1.AzureAdFederatedCredential
	}
	issuerSubject struct {
		issuer  string
		subject string
	}
)

func diff(existing []msgraph.FederatedIdentityCredential, desired []v1.AzureAdFederatedCredential) (changes, error) {
	desiredByName := make(map[string]v1.AzureAdFederatedCredential, len(desired))
	for _, credential := range desired {
		desiredByName[credential.Name] = credential
	}

	existingByName := make(map[string]msgraph.FederatedIdentityCredential, len(existing))
	ownerOfPair := make(map[issuerSubject]string, len(existing))
	for _, credential := range existing {
		name := stringValue(credential.Name)
		existingByName[name] = credential
		ownerOfPair[issuerSubject{issuer: stringValue(credential.Issuer), subject: stringValue(credential.Subject)}] = name
	}

	result := changes{}
	recreate := make(map[string]struct{})
	for _, credential := range desired {
		current, found := existingByName[credential.Name]
		if !found {
			result.toCreate = append(result.toCreate, credential)
			continue
		}
		if matches(current, credential) {
			continue
		}
		if current.ID == nil {
			return changes{}, fmt.Errorf("updating federated identity credential %q: missing Graph ID", credential.Name)
		}

		owner, held := ownerOfPair[issuerSubject{issuer: credential.Issuer, subject: credential.Subject}]
		_, ownerRemains := desiredByName[owner]
		if held && owner != credential.Name && ownerRemains {
			recreate[credential.Name] = struct{}{}
			result.toCreate = append(result.toCreate, credential)
			continue
		}
		result.toUpdate = append(result.toUpdate, credentialUpdate{id: *current.ID, desired: credential})
	}

	for _, credential := range existing {
		name := stringValue(credential.Name)
		_, remains := desiredByName[name]
		_, replaced := recreate[name]
		if remains && !replaced {
			continue
		}
		if credential.ID == nil {
			return changes{}, fmt.Errorf("deleting federated identity credential %q: missing Graph ID", name)
		}
		result.toDelete = append(result.toDelete, credential)
	}

	return result, nil
}

func fromDesired(desired v1.AzureAdFederatedCredential) msgraph.FederatedIdentityCredential {
	return msgraph.FederatedIdentityCredential{
		Name:      new(desired.Name),
		Audiences: []string{desired.Audience},
		Issuer:    new(desired.Issuer),
		Subject:   new(desired.Subject),
	}
}

func matches(existing msgraph.FederatedIdentityCredential, desired v1.AzureAdFederatedCredential) bool {
	return stringValue(existing.Issuer) == desired.Issuer &&
		stringValue(existing.Subject) == desired.Subject &&
		len(existing.Audiences) == 1 && existing.Audiences[0] == desired.Audience
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
