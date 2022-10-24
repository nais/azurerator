package team

import (
	"fmt"
	"strings"

	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/transaction"
)

type Owners interface {
	Process(tx transaction.Transaction) error
}

type owners struct {
	Client
}

func newOwners(client Client) Owners {
	return owners{Client: client}
}

func (o owners) Process(tx transaction.Transaction) error {
	owners, err := o.get(tx)
	if err != nil {
		return err
	}

	if err = o.Application().Owners().Process(tx, owners); err != nil {
		return fmt.Errorf("processing application owners: %w", err)
	}

	if err = o.ServicePrincipal().Owners().Process(tx, owners); err != nil {
		return fmt.Errorf("processing service principal owners: %w", err)
	}
	return nil
}

func (o owners) get(tx transaction.Transaction) ([]msgraph.DirectoryObject, error) {
	owners := make([]msgraph.DirectoryObject, 0)

	group, err := o.getTeamGroup(tx)
	if err != nil {
		return owners, err
	}
	if group == nil {
		return owners, nil
	}
	groupId := (string)(*group.PrincipalID)

	owners, err = o.Groups().GetOwnersFor(tx.Ctx, groupId)
	if err != nil {
		return owners, err
	}

	return owners, nil
}

func (o owners) getTeamGroup(tx transaction.Transaction) (*msgraph.AppRoleAssignment, error) {
	var group *msgraph.AppRoleAssignment
	groups, err := o.Team().Groups().Get(tx)
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
