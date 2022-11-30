package group

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	cache "github.com/Code-Hex/go-generics-cache"
	"github.com/nais/msgraph.go/jsonx"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application/approle"
	"github.com/nais/azureator/pkg/azure/client/serviceprincipal"
	"github.com/nais/azureator/pkg/azure/permissions"
	"github.com/nais/azureator/pkg/azure/resource"
	"github.com/nais/azureator/pkg/transaction"
)

const cacheExpiration = 24 * time.Hour

var groupCache = cache.New[azure.ObjectId, *msgraph.Group]()

var (
	BadRequestError = errors.New("BadRequest")
)

type Groups interface {
	Process(tx transaction.Transaction) error
}

type Client interface {
	azure.RuntimeClient
	AppRoleAssignments(tx transaction.Transaction, targetId azure.ObjectId) serviceprincipal.AppRoleAssignments
}

type group struct {
	Client
}

func NewGroup(client Client) Groups {
	return group{Client: client}
}

func (g group) Process(tx transaction.Transaction) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	groups, err := g.getGroups(tx)
	if err != nil {
		return err
	}

	// TODO(tronghn): if there exists an AppRole where AllowedMemberTypes includes "User", then we cannot use the default AppRole `00000000-0000-0000-0000-000000000000`.
	//  Should ensure that a default group role is created and used for this case.
	roles := make(permissions.Permissions)
	roles.Add(permissions.FromAppRole(approle.DefaultGroupRole()))

	err = g.AppRoleAssignments(tx, servicePrincipalId).
		ProcessForGroups(groups, roles)
	if err != nil {
		return fmt.Errorf("updating app roles for groups: %w", err)
	}

	return nil
}

func (g group) getGroups(tx transaction.Transaction) (resource.Resources, error) {
	groups, err := g.getGroupsFromClaims(tx)
	if err != nil {
		return nil, fmt.Errorf("mapping group claims to resources: %w", err)
	}

	undefinedAllUsersGroupID := len(g.Config().Features.GroupsAssignment.AllUsersGroupId) == 0
	appRoleAssignmentNotRequired := !g.Config().Features.AppRoleAssignmentRequired.Enabled

	if undefinedAllUsersGroupID || appRoleAssignmentNotRequired {
		return groups, nil
	}

	allowAllUsersEnabled := tx.Instance.Spec.AllowAllUsers != nil && *tx.Instance.Spec.AllowAllUsers == true
	if allowAllUsersEnabled {
		allUsersGroups, err := g.getAllUsersGroups(tx)
		if err != nil {
			return nil, fmt.Errorf("mapping all-users group to resources: %w", err)
		}

		if allUsersGroups != nil {
			groups.AddAll(allUsersGroups...)
		}
	}

	return groups, nil
}

func (g group) getGroupsFromClaims(tx transaction.Transaction) (resource.Resources, error) {
	seen := make(map[string]bool)
	resources := make(resource.Resources, 0)

	if tx.Instance.Spec.Claims == nil || len(tx.Instance.Spec.Claims.Groups) == 0 {
		return resources, nil
	}

	for _, group := range tx.Instance.Spec.Claims.Groups {
		exists, groupResult, err := g.getById(tx, group.ID)
		if err != nil {
			if errors.Is(err, BadRequestError) {
				tx.Logger.Warnf("groups: skipping assignment %s: %+v", group, err)
				continue
			}
			return nil, fmt.Errorf("getting group '%s': %w", group, err)
		}

		if !exists {
			tx.Logger.Debugf("groups: skipping assignment: '%s' does not exist", group.ID)
			continue
		}

		if !seen[group.ID] {
			resources = append(resources, g.mapToResource(*groupResult))
			seen[group.ID] = true
		}
	}

	return resources, nil
}

func (g group) getAllUsersGroups(tx transaction.Transaction) ([]resource.Resource, error) {
	allUsersGroupIDs := g.Config().Features.GroupsAssignment.AllUsersGroupId
	groups := make([]resource.Resource, len(allUsersGroupIDs))

	for _, id := range allUsersGroupIDs {
		exists, groupResult, err := g.getById(tx, id)
		if err != nil {
			return nil, fmt.Errorf("getting group '%s': %w", allUsersGroupIDs, err)
		}

		if !exists {
			return nil, fmt.Errorf("group '%s' does not exist: %w", allUsersGroupIDs, err)
		}

		groups = append(groups, g.mapToResource(*groupResult))
	}
	return groups, nil
}

func (g group) getById(tx transaction.Transaction, id azure.ObjectId) (bool, *msgraph.Group, error) {
	if len(id) == 0 {
		return false, nil, nil
	}
	if val, found := groupCache.Get(id); found {
		tx.Logger.Debugf("groups: cache hit for '%s'", id)
		return true, val, nil
	}
	tx.Logger.Debugf("groups: cache miss for '%s'", id)

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

	if res.StatusCode == 400 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return false, nil, fmt.Errorf("reading server response: %w", err)
		}

		return false, nil, fmt.Errorf("%w: %s", BadRequestError, body)
	}

	var group *msgraph.Group
	exists, err := g.decodeJsonResponseForGetRequest(res, &group)
	if err != nil {
		return false, nil, fmt.Errorf("decoding json response: %w", err)
	}

	if !exists || group == nil || group.ID == nil || group.DisplayName == nil {
		return false, nil, nil
	}

	groupCache.Set(id, group, cache.WithExpiration(cacheExpiration))
	return true, group, nil
}

func (g group) toGetRequestWithContext(ctx context.Context, r *msgraph.GroupRequest) (*http.Request, error) {
	req, err := r.NewJSONRequest("GET", "", nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	return req, nil
}

func (g group) decodeJsonResponseForGetRequest(res *http.Response, obj any) (bool, error) {
	switch res.StatusCode {
	case http.StatusOK, http.StatusCreated:
		if obj == nil {
			return false, nil
		}

		err := jsonx.NewDecoder(res.Body).Decode(obj)
		if err != nil {
			return false, err
		}
		return true, nil
	case http.StatusNoContent, http.StatusNotFound:
		return false, nil
	default:
		b, _ := io.ReadAll(res.Body)
		errRes := &msgraph.ErrorResponse{Response: res}
		err := jsonx.Unmarshal(b, errRes)
		if err != nil {
			return false, fmt.Errorf("%s: %s", res.Status, string(b))
		}
		return false, errRes
	}
}

func (g group) mapToResource(group msgraph.Group) resource.Resource {
	return resource.Resource{
		Name:          *group.DisplayName,
		ClientId:      "",
		ObjectId:      *group.ID,
		PrincipalType: resource.PrincipalTypeGroup,
	}
}
