package team

import (
	"fmt"

	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/client/approleassignment"
	"github.com/nais/azureator/pkg/azure/transaction"
)

type Groups interface {
	Get(tx transaction.Transaction) (approleassignment.List, error)
}

type groups struct {
	Client
}

func newGroups(client Client) Groups {
	return groups{Client: client}
}

func (g groups) Get(tx transaction.Transaction) (approleassignment.List, error) {
	groups := make([]msgraph.AppRoleAssignment, 0)
	targetId := g.Config().Features.TeamsManagement.ServicePrincipalId

	if len(targetId) == 0 {
		return groups, nil
	}

	groups, err := g.AppRoleAssignments(tx, targetId).GetAllGroups()
	if err != nil {
		return groups, fmt.Errorf("failed to get assigned groups for teams management service principal: %w", err)
	}

	return groups, nil
}
