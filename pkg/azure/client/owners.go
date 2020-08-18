package client

import (
	"strings"

	"github.com/nais/azureator/pkg/azure"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type teamowners struct {
	client
}

func (c client) teamowners() teamowners {
	return teamowners{c}
}

func (to teamowners) register(tx azure.Transaction, objectId azure.ObjectId, spId azure.ServicePrincipalId) error {
	owners, err := to.get(tx)
	if err != nil {
		return err
	}
	if len(owners) == 0 {
		return nil
	}
	if err = to.application().owners().register(tx.Ctx, objectId, owners); err != nil {
		return err
	}
	if err = to.servicePrincipal().owners().register(tx.Ctx, spId, owners); err != nil {
		return err
	}
	return nil
}

func (to teamowners) get(tx azure.Transaction) ([]msgraph.DirectoryObject, error) {
	owners := make([]msgraph.DirectoryObject, 0)
	group, err := to.getTeamGroup(tx)
	if err != nil {
		return owners, err
	}
	if group == nil {
		return owners, nil
	}
	groupId := (string)(*group.PrincipalID)
	owners, err = to.group().getOwnersFor(tx.Ctx, groupId)
	if err != nil {
		return owners, err
	}
	return owners, nil
}

func (to teamowners) update(tx azure.Transaction) error {
	objectId := tx.Instance.Status.ObjectId
	servicePrincipalId := tx.Instance.Status.ServicePrincipalId

	if err := to.register(tx, objectId, servicePrincipalId); err != nil {
		return err
	}
	if err := to.revoke(tx); err != nil {
		return err
	}
	return nil
}

func (to teamowners) revoke(tx azure.Transaction) error {
	objectId := tx.Instance.Status.ObjectId
	servicePrincipalId := tx.Instance.Status.ServicePrincipalId
	if err := to.application().owners().revoke(tx, objectId); err != nil {
		return err
	}
	if err := to.servicePrincipal().owners().revoke(tx, servicePrincipalId); err != nil {
		return err
	}
	return nil
}

func (to teamowners) getTeamGroup(tx azure.Transaction) (*msgraphbeta.AppRoleAssignment, error) {
	var group *msgraphbeta.AppRoleAssignment
	groups, err := to.teams().get(tx.Ctx)
	if err != nil {
		return group, err
	}
	teamName := strings.ToLower(tx.Instance.Namespace)
	for _, g := range groups {
		groupName := strings.ToLower(*g.PrincipalDisplayName)
		if groupName == teamName {
			return &g, nil
		}
	}
	return group, nil
}
