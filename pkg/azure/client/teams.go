package client

import (
	"context"
	"fmt"

	msgraph "github.com/nais/msgraph.go/v1.0"
)

type teams struct {
	client
}

func (c client) teams() teams {
	return teams{c}
}

func (t teams) get(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	groups := make([]msgraph.AppRoleAssignment, 0)
	if len(t.config.Features.TeamsManagement.ServicePrincipalId) == 0 {
		return groups, nil
	}
	targetId := t.config.Features.TeamsManagement.ServicePrincipalId
	groups, err := t.appRoleAssignments("", targetId).getAllGroups(ctx)
	if err != nil {
		return groups, fmt.Errorf("failed to get assigned groups for teams management service principal: %w", err)
	}
	return groups, nil
}
