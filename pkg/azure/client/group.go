package client

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type groups struct {
	client
}

func (c client) groups() groups {
	return groups{c}
}

func (g groups) getOwnersFor(ctx context.Context, groupId string) ([]msgraph.DirectoryObject, error) {
	owners, err := g.graphClient.Groups().ID(groupId).Owners().Request().GetN(ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return owners, fmt.Errorf("failed to fetch owners for group: %w", err)
	}
	return owners, nil
}

func (g groups) process(tx azure.Transaction) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	groups, err := g.mapToResources(tx)
	if err != nil {
		return fmt.Errorf("looking up groups: %w", err)
	}

	err = g.appRoleAssignments(msgraphbeta.UUID(DefaultGroupRoleId), servicePrincipalId).
		processForGroups(tx, groups)
	if err != nil {
		return fmt.Errorf("updating app roles for groups: %w", err)
	}
	return nil
}

func (g groups) exists(tx azure.Transaction, id azure.ObjectId) (bool, error) {
	groups, err := g.getAll(tx, util.FilterById(id))
	if err != nil {
		return false, fmt.Errorf("chceking existence of group: %w", err)
	}
	return len(groups) > 0, nil
}

func (g groups) getAll(tx azure.Transaction, filters ...azure.Filter) ([]msgraph.Group, error) {
	req := g.graphClient.Groups().Request()
	filter := util.MapFiltersToFilter(filters)
	req.Filter(filter)

	groups, err := req.GetN(tx.Ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return nil, fmt.Errorf("getting groups with filter '%s': %w", filter, err)
	}

	return groups, nil
}

func (g groups) getById(tx azure.Transaction, id azure.ObjectId) (*msgraph.Group, error) {
	group, err := g.graphClient.Groups().ID(id).Request().Get(tx.Ctx)
	if err != nil {
		return nil, fmt.Errorf("getting group '%s': %s", id, err)
	}

	return group, nil
}

func (g groups) mapToResources(tx azure.Transaction) ([]azure.Resource, error) {
	resources := make([]azure.Resource, 0)

	if tx.Instance.Spec.Claims == nil {
		return resources, nil
	}

	for _, group := range tx.Instance.Spec.Claims.Groups {
		exists, err := g.exists(tx, group)
		if err != nil {
			return nil, fmt.Errorf("checking if group exists: %w", err)
		}

		if !exists {
			continue
		}

		groupResult, err := g.getById(tx, group)
		if err != nil {
			return nil, fmt.Errorf("getting group: %w", err)
		}

		resources = append(resources, azure.Resource{
			Name:          *groupResult.DisplayName,
			ClientId:      "",
			ObjectId:      *groupResult.ID,
			PrincipalType: azure.PrincipalTypeGroup,
		})
	}
	return resources, nil
}
