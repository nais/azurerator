package client

import (
	"fmt"
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

	owners, err = to.groups().getOwnersFor(tx.Ctx, groupId)
	if err != nil {
		return owners, err
	}

	return owners, nil
}

func (to teamowners) process(tx azure.Transaction) error {
	owners, err := to.get(tx)
	if err != nil {
		return err
	}

	if err = to.application().owners().process(tx, owners); err != nil {
		return fmt.Errorf("processing application owners: %w", err)
	}

	if err = to.servicePrincipal().owners().process(tx, owners); err != nil {
		return fmt.Errorf("processing service principal owners: %w", err)
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
