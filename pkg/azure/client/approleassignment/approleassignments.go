package approleassignment

import (
	"context"
	"fmt"

	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/resource"
)

type appRoleAssignments struct {
	azure.RuntimeClient
	targetId  azure.ObjectId
	logFields log.Fields
}

func NewAppRoleAssignmentsNoRoleId(client azure.RuntimeClient, targetId azure.ObjectId) azure.AppRoleAssignments {
	return appRoleAssignments{
		RuntimeClient: client,
		targetId:      targetId,
		logFields: log.Fields{
			"targetId": targetId,
		},
	}
}

func (a appRoleAssignments) GetAll(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	assignments, err := a.Request().GetN(ctx, a.MaxNumberOfPagesToFetch())
	if err != nil {
		return nil, fmt.Errorf("looking up AppRole assignments for service principal '%s': %w", a.TargetId(), err)
	}
	return assignments, nil
}

func (a appRoleAssignments) GetAllGroups(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	assignments, err := a.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	groups := filterByType(assignments, resource.PrincipalTypeGroup)
	return groups, nil
}

func (a appRoleAssignments) GetAllServicePrincipals(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	assignments, err := a.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	servicePrincipals := filterByType(assignments, resource.PrincipalTypeServicePrincipal)
	return servicePrincipals, nil
}

func (a appRoleAssignments) Request() *msgraph.ServicePrincipalAppRoleAssignedToCollectionRequest {
	return a.GraphClient().ServicePrincipals().ID(a.TargetId()).AppRoleAssignedTo().Request()
}

func (a appRoleAssignments) TargetId() azure.ObjectId {
	return a.targetId
}

func (a appRoleAssignments) LogFields() log.Fields {
	return a.logFields
}

func filterByType(assignments []msgraph.AppRoleAssignment, principalType resource.PrincipalType) []msgraph.AppRoleAssignment {
	filtered := make([]msgraph.AppRoleAssignment, 0)
	for _, assignment := range assignments {
		if resource.PrincipalType(*assignment.PrincipalType) == principalType {
			filtered = append(filtered, assignment)
		}
	}
	return filtered
}
