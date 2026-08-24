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
	logger := tx.Logger.WithField("subsystem", "federatedcredential")
	credentials := f.GraphClient().Applications().ID(objectID).FederatedIdentityCredentials()

	existing, err := credentials.Request().GetN(tx.Ctx, f.MaxNumberOfPagesToFetch())
	if err != nil {
		return fmt.Errorf("listing federated identity credentials for application: %w", err)
	}

	changes, err := diff(existing, tx.Instance.Spec.FederatedCredentials)
	if err != nil {
		return err
	}

	// Delete credentials first to allow transferring ownership of issuer-subject pairs to another credential
	for _, toDelete := range changes.toDelete {
		time.Sleep(f.DelayIntervalBetweenModifications())
		if err := credentials.ID(*toDelete.ID).Request().Delete(tx.Ctx); err != nil {
			return fmt.Errorf("deleting federated identity credential %q: %w", valueOrEmpty(toDelete.Name), err)
		}
		logger.Debugf("deleted federated identity credential %q", valueOrEmpty(toDelete.Name))
	}

	for _, toUpdate := range changes.toUpdate {
		time.Sleep(f.DelayIntervalBetweenModifications())
		credential := fromDesired(toUpdate.AzureAdFederatedCredential)
		// Names are immutable and must not be included in a Graph PATCH request.
		credential.Name = nil
		if err := credentials.ID(toUpdate.id).Request().Update(tx.Ctx, &credential); err != nil {
			return fmt.Errorf("updating federated identity credential %q: %w", toUpdate.Name, err)
		}
		logger.Debugf("updated federated identity credential %q", toUpdate.Name)
	}

	for _, toCreate := range changes.toCreate {
		time.Sleep(f.DelayIntervalBetweenModifications())
		credential := fromDesired(toCreate)
		if _, err := credentials.Request().Add(tx.Ctx, &credential); err != nil {
			return fmt.Errorf("creating federated identity credential %q: %w", toCreate.Name, err)
		}
		logger.Debugf("created federated identity credential %q", toCreate.Name)
	}

	return nil
}

type (
	credentialOperations struct {
		toCreate []v1.AzureAdFederatedCredential
		toUpdate []credentialUpdate
		toDelete []msgraph.FederatedIdentityCredential
	}
	credentialDiff struct {
		existing []msgraph.FederatedIdentityCredential
		desired  []v1.AzureAdFederatedCredential
	}
	credentialUpdate struct {
		id string
		v1.AzureAdFederatedCredential
	}
)

func diff(existing []msgraph.FederatedIdentityCredential, desired []v1.AzureAdFederatedCredential) (credentialOperations, error) {
	return credentialDiff{existing: existing, desired: desired}.calculate()
}

func (d credentialDiff) calculate() (credentialOperations, error) {
	// Graph requires issuer-subject pairs to be unique. Recreate a credential when its
	// desired pair belongs to another retained name, so deletions break update dependencies.
	ops := credentialOperations{}

	for _, desired := range d.desired {
		existing, found := d.findExisting(desired.Name)
		if !found {
			ops.toCreate = append(ops.toCreate, desired)
			continue
		}
		if matches(existing, desired) {
			continue
		}
		if d.requiresRecreation(desired) {
			ops.toDelete = append(ops.toDelete, existing)
			ops.toCreate = append(ops.toCreate, desired)
			continue
		}
		ops.toUpdate = append(ops.toUpdate, credentialUpdate{id: *existing.ID, AzureAdFederatedCredential: desired})
	}

	for _, existing := range d.existing {
		name := valueOrEmpty(existing.Name)
		_, stillDesired := d.findDesired(name)
		if stillDesired {
			continue
		}
		ops.toDelete = append(ops.toDelete, existing)
	}

	return ops, nil
}

func (d credentialDiff) findDesired(name string) (v1.AzureAdFederatedCredential, bool) {
	for _, credential := range d.desired {
		if credential.Name == name {
			return credential, true
		}
	}
	return v1.AzureAdFederatedCredential{}, false
}

func (d credentialDiff) findExisting(name string) (msgraph.FederatedIdentityCredential, bool) {
	for _, credential := range d.existing {
		if valueOrEmpty(credential.Name) == name {
			return credential, true
		}
	}
	return msgraph.FederatedIdentityCredential{}, false
}

func (d credentialDiff) requiresRecreation(credential v1.AzureAdFederatedCredential) bool {
	ownerOf := func(issuer, subject string) (string, bool) {
		for _, existing := range d.existing {
			if valueOrEmpty(existing.Issuer) == issuer && valueOrEmpty(existing.Subject) == subject {
				return valueOrEmpty(existing.Name), true
			}
		}
		return "", false
	}

	ownerName, found := ownerOf(credential.Issuer, credential.Subject)
	ownedByAnotherCredential := found && ownerName != credential.Name
	_, ownerStillDesired := d.findDesired(ownerName)

	return ownedByAnotherCredential && ownerStillDesired
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
	return valueOrEmpty(existing.Issuer) == desired.Issuer &&
		valueOrEmpty(existing.Subject) == desired.Subject &&
		len(existing.Audiences) == 1 && existing.Audiences[0] == desired.Audience
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
