package client

import (
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/labels"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type owners struct {
	client
}

func (c client) owners() owners {
	return owners{c}
}

func (o owners) register(tx azure.Transaction, objectId azure.ObjectId, spId azure.ServicePrincipalId) error {
	owners, err := o.get(tx)
	if err != nil {
		return err
	}
	if len(owners) == 0 {
		return nil
	}
	if err = o.application().registerOwners(tx.Ctx, objectId, owners); err != nil {
		return err
	}
	if err = o.servicePrincipal().registerOwners(tx.Ctx, spId, owners); err != nil {
		return err
	}
	return nil
}

func (o owners) get(tx azure.Transaction) ([]msgraph.DirectoryObject, error) {
	owners := make([]msgraph.DirectoryObject, 0)
	group, err := o.getTeamGroup(tx)
	if err != nil {
		return owners, err
	}
	if group == nil {
		return owners, nil
	}
	groupId := (string)(*group.PrincipalID)
	owners, err = o.group().getOwnersFor(tx.Ctx, groupId)
	if err != nil {
		return owners, err
	}
	return owners, nil
}

func (o owners) update(tx azure.Transaction) error {
	objectId := tx.Instance.Status.ObjectId
	servicePrincipalId := tx.Instance.Status.ServicePrincipalId

	if err := o.register(tx, objectId, servicePrincipalId); err != nil {
		return err
	}
	if err := o.revoke(tx); err != nil {
		return err
	}
	return nil
}

func (o owners) revoke(tx azure.Transaction) error {
	objectId := tx.Instance.Status.ObjectId
	servicePrincipalId := tx.Instance.Status.ServicePrincipalId
	if err := o.application().revokeOwners(tx, objectId); err != nil {
		return err
	}
	if err := o.servicePrincipal().revokeOwners(tx, servicePrincipalId); err != nil {
		return err
	}
	return nil
}

func (o owners) getTeamGroup(tx azure.Transaction) (*msgraphbeta.AppRoleAssignment, error) {
	var group *msgraphbeta.AppRoleAssignment
	groups, err := o.teams().get(tx.Ctx)
	if err != nil {
		return group, err
	}
	teamName := tx.Instance.Labels[labels.TeamLabelKey]
	for _, g := range groups {
		if *g.PrincipalDisplayName == teamName {
			return &g, nil
		}
	}
	return group, nil
}
