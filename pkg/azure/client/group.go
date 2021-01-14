package client

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/jsonx"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	"io/ioutil"
	"net/http"
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

func (g groups) getById(tx azure.Transaction, id azure.ObjectId) (bool, *msgraph.Group, error) {
	r := g.graphClient.Groups().ID(id).Request()

	req, err := g.toGetRequestWithContext(tx.Ctx, r)
	if err != nil {
		return false, nil, fmt.Errorf("to json request: %w", err)
	}

	res, err := g.httpClient.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("performing http request: %w", err)
	}

	defer res.Body.Close()

	var group *msgraph.Group
	exists, err := g.decodeJsonResponse(res, &group)
	if err != nil {
		return exists, nil, fmt.Errorf("decoding json response: %w", err)
	}

	return exists, group, nil
}

func (g groups) mapToResources(tx azure.Transaction) ([]azure.Resource, error) {
	resources := make([]azure.Resource, 0)

	if tx.Instance.Spec.Claims == nil || len(tx.Instance.Spec.Claims.Groups) == 0 {
		// TODO: assign default group with "all users"
		return resources, nil
	}

	for _, group := range tx.Instance.Spec.Claims.Groups {
		exists, groupResult, err := g.getById(tx, group.ID)
		if err != nil {
			return nil, fmt.Errorf("getting group '%s': %w", group, err)
		}

		if !exists {
			tx.Log.Debugf("skipping Group assignment: '%s' does not exist", group)
			continue
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

func (g groups) toGetRequestWithContext(ctx context.Context, r *msgraph.GroupRequest) (*http.Request, error) {
	req, err := r.NewJSONRequest("GET", "", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	return req, nil
}

func (g groups) decodeJsonResponse(res *http.Response, obj interface{}) (bool, error) {
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
