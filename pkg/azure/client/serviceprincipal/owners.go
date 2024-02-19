package serviceprincipal

import (
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/directoryobject"
	"github.com/nais/azureator/pkg/transaction"
	msgraph "github.com/nais/msgraph.go/v1.0"
)

type Owners interface {
	Process(tx transaction.Transaction, owner azure.ServicePrincipalId) error
}

type owners struct {
	azure.RuntimeClient
}

func newOwners(client azure.RuntimeClient) Owners {
	return &owners{RuntimeClient: client}
}

func (o owners) Process(tx transaction.Transaction, owner azure.ServicePrincipalId) error {
	existing, err := o.get(tx)
	if err != nil {
		return err
	}

	if directoryobject.ContainsOwner(existing, owner) {
		return nil
	}

	return o.add(tx, owner)
}

func (o owners) get(tx transaction.Transaction) ([]msgraph.DirectoryObject, error) {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()
	owners, err := o.GraphClient().ServicePrincipals().ID(servicePrincipalId).Owners().Request().GetN(tx.Ctx, o.MaxNumberOfPagesToFetch())
	if err != nil {
		return owners, fmt.Errorf("listing owners for service principal: %w", err)
	}
	return owners, nil
}

func (o owners) add(tx transaction.Transaction, owner azure.ServicePrincipalId) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	body := directoryobject.ToOwnerPayload(owner)
	req := o.GraphClient().ServicePrincipals().ID(servicePrincipalId).Owners().Request()

	err := req.JSONRequest(tx.Ctx, "POST", "/$ref", body, nil)
	if err != nil {
		return fmt.Errorf("adding owner %q to service principal: %w", owner, err)
	}

	tx.Logger.Infof("assigned owner %q to service principal", owner)
	return nil
}
