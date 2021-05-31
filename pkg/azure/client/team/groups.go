package team

import (
	"context"
	"fmt"

	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
)

type groups struct {
	azure.RuntimeClient
}

func newGroups(client azure.RuntimeClient) azure.TeamGroups {
	return groups{RuntimeClient: client}
}

func (g groups) Get(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	groups := make([]msgraph.AppRoleAssignment, 0)
	targetId := g.Config().Features.TeamsManagement.ServicePrincipalId

	if len(targetId) == 0 {
		return groups, nil
	}

	groups, err := g.AppRoleAssignmentsNoRoleId(targetId).GetAllGroups(ctx)
	if err != nil {
		return groups, fmt.Errorf("failed to get assigned groups for teams management service principal: %w", err)
	}

	return groups, nil
}
