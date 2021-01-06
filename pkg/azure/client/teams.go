package client

import (
	"context"
	"fmt"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
)

type teams struct {
	client
}

func (c client) teams() teams {
	return teams{c}
}

func (t teams) get(ctx context.Context) ([]msgraphbeta.AppRoleAssignment, error) {
	groups := make([]msgraphbeta.AppRoleAssignment, 0)
	if len(t.config.Features.TeamsManagement.ServicePrincipalId) == 0 {
		return groups, nil
	}
	groups, err := t.appRoleAssignments().getGroups(ctx, t.config.Features.TeamsManagement.ServicePrincipalId)
	if err != nil {
		return groups, fmt.Errorf("failed to get assigned groups for teams management service principal: %w", err)
	}
	return groups, nil
}
