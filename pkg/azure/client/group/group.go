package group

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/nais/msgraph.go/jsonx"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/approle"
)

type group struct {
	azure.RuntimeClient
}

func NewGroup(runtimeClient azure.RuntimeClient) azure.Groups {
	return group{RuntimeClient: runtimeClient}
}

func (g group) GetOwnersFor(ctx context.Context, groupId string) ([]msgraph.DirectoryObject, error) {
	owners, err := g.GraphClient().Groups().ID(groupId).Owners().Request().GetN(ctx, g.MaxNumberOfPagesToFetch())
	if err != nil {
		return owners, fmt.Errorf("failed to fetch owners for group: %w", err)
	}
	return owners, nil
}

func (g group) Process(tx azure.Transaction) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	groups, err := g.mapGroupClaimsToResources(tx)
	if err != nil {
		return fmt.Errorf("mapping group claims to resources: %w", err)
	}

	if len(groups) == 0 {
		groups, err = g.mapAllUserGroupToResources(tx)

		if err != nil {
			return fmt.Errorf("mapping all-users group to resources: %w", err)
		}
	}

	err = g.AppRoleAssignments(msgraph.UUID(approle.DefaultGroupRoleId), servicePrincipalId).
		ProcessForGroups(tx, groups)
	if err != nil {
		return fmt.Errorf("updating app roles for groups: %w", err)
	}

	return nil
}

func (g group) getById(tx azure.Transaction, id azure.ObjectId) (bool, *msgraph.Group, error) {
	r := g.GraphClient().Groups().ID(id).Request()

	req, err := g.toGetRequestWithContext(tx.Ctx, r)
	if err != nil {
		return false, nil, fmt.Errorf("to json request: %w", err)
	}

	res, err := g.HttpClient().Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("performing http request: %w", err)
	}

	defer res.Body.Close()

	var group *msgraph.Group
	exists, err := g.decodeJsonResponseForGetRequest(res, &group)
	if err != nil {
		return exists, nil, fmt.Errorf("decoding json response: %w", err)
	}

	return exists, group, nil
}

func (g group) mapGroupClaimsToResources(tx azure.Transaction) ([]azure.Resource, error) {
	resources := make([]azure.Resource, 0)

	if tx.Instance.Spec.Claims == nil || len(tx.Instance.Spec.Claims.Groups) == 0 {
		return resources, nil
	}

	for _, group := range tx.Instance.Spec.Claims.Groups {
		exists, groupResult, err := g.getById(tx, group.ID)
		if err != nil {
			return nil, fmt.Errorf("getting group '%s': %w", group, err)
		}

		if !exists {
			tx.Log.Debugf("skipping Groups assignment: '%s' does not exist", group.ID)
			continue
		}

		resources = append(resources, g.mapToResource(*groupResult))
	}

	return resources, nil
}

func (g group) mapAllUserGroupToResources(tx azure.Transaction) ([]azure.Resource, error) {
	allUsersGroupID := g.Config().Features.GroupsAssignment.AllUsersGroupId

	exists, groupResult, err := g.getById(tx, allUsersGroupID)
	if err != nil {
		return nil, fmt.Errorf("getting all-users group '%s': %w", allUsersGroupID, err)
	}

	if !exists {
		return nil, fmt.Errorf("all-users group '%s' does not exist: %w", allUsersGroupID, err)
	}

	return []azure.Resource{g.mapToResource(*groupResult)}, nil
}

func (g group) toGetRequestWithContext(ctx context.Context, r *msgraph.GroupRequest) (*http.Request, error) {
	req, err := r.NewJSONRequest("GET", "", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	return req, nil
}

func (g group) decodeJsonResponseForGetRequest(res *http.Response, obj interface{}) (bool, error) {
	switch res.StatusCode {
	case http.StatusOK, http.StatusCreated:
		if obj != nil {
			err := jsonx.NewDecoder(res.Body).Decode(obj)
			if err != nil {
				return false, err
			}
		}
		return true, nil
	case http.StatusNoContent:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		b, _ := ioutil.ReadAll(res.Body)
		errRes := &msgraph.ErrorResponse{Response: res}
		err := jsonx.Unmarshal(b, errRes)
		if err != nil {
			return false, fmt.Errorf("%s: %s", res.Status, string(b))
		}
		return true, errRes
	}
}

func (g group) mapToResource(group msgraph.Group) azure.Resource {
	return azure.Resource{
		Name:          *group.DisplayName,
		ClientId:      "",
		ObjectId:      *group.ID,
		PrincipalType: azure.PrincipalTypeGroup,
	}
}
