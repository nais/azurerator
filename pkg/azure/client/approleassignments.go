package client

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/approleassignment"
	msgraph "github.com/nais/msgraph.go/v1.0"
)

type appRoleAssignments struct {
	client
	roleId    msgraph.UUID
	targetId  azure.ObjectId
	logFields log.Fields
}

func (c client) appRoleAssignments(roleId msgraph.UUID, targetId azure.ObjectId) appRoleAssignments {
	return appRoleAssignments{
		client:   c,
		roleId:   roleId,
		targetId: targetId,
		logFields: log.Fields{
			"roleId":   roleId,
			"targetId": targetId,
		},
	}
}

func (a appRoleAssignments) request() *msgraph.ServicePrincipalAppRoleAssignedToCollectionRequest {
	return a.graphClient.ServicePrincipals().ID(a.targetId).AppRoleAssignedTo().Request()
}

func (a appRoleAssignments) assignFor(tx azure.Transaction, assignees []azure.Resource, principalType azure.PrincipalType) ([]msgraph.AppRoleAssignment, error) {
	assignments := make([]msgraph.AppRoleAssignment, 0)

	existing, err := a.getAll(tx.Ctx)
	if err != nil {
		return nil, fmt.Errorf("looking up existing AppRole assignments: %w", err)
	}

	for _, assignee := range assignees {
		logFields := a.logFields
		logFields["assigneeClientId"] = assignee.ClientId
		logFields["assigneeObjectId"] = assignee.ObjectId

		if len(assignee.ObjectId) == 0 {
			tx.Log.WithFields(logFields).Debugf("skipping AppRole assignment: object ID for %s '%s' is not set", assignee.PrincipalType, assignee.Name)
			continue
		}

		assignment := a.toAssignment(assignee.ObjectId, principalType)
		if assignmentInAssignments(*assignment, existing) {
			tx.Log.WithFields(logFields).Infof("skipping AppRole assignment: already assigned for %s '%s'", assignee.PrincipalType, assignee.Name)
			assignments = append(assignments, *assignment)
			continue
		}

		tx.Log.WithFields(logFields).Debugf("assigning AppRole for %s '%s'...", principalType, assignee.Name)
		result, err := a.request().Add(tx.Ctx, assignment)
		if err != nil {
			return nil, fmt.Errorf("assigning AppRole for %s '%s' (%s) to target service principal ID '%s': %w", assignee.PrincipalType, assignee.Name, assignee.ObjectId, a.targetId, err)
		}

		if result.AppRoleID != nil && result.ResourceID != nil && result.PrincipalID != nil {
			tx.Log.WithFields(logFields).Infof("successfully assigned AppRole for %s '%s'", assignee.PrincipalType, assignee.Name)
			assignments = append(assignments, *result)
		}
	}
	return assignments, nil
}

func (a appRoleAssignments) getAll(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	assignments, err := a.request().GetN(ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return nil, fmt.Errorf("looking up AppRole assignments for service principal '%s': %w", a.targetId, err)
	}
	return assignments, nil
}

func (a appRoleAssignments) getAllGroups(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	assignments, err := a.getAll(ctx)
	if err != nil {
		return nil, err
	}
	groups := filterByType(assignments, azure.PrincipalTypeGroup)
	return groups, nil
}

func (a appRoleAssignments) getAllServicePrincipals(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	assignments, err := a.getAll(ctx)
	if err != nil {
		return nil, err
	}
	servicePrincipals := filterByType(assignments, azure.PrincipalTypeServicePrincipal)
	return servicePrincipals, nil
}

func (a appRoleAssignments) revokeFor(tx azure.Transaction, revoked []msgraph.AppRoleAssignment, principalType azure.PrincipalType) error {
	for _, r := range revoked {
		logFields := a.logFields
		logFields["assigneeObjectId"] = *r.PrincipalID

		tx.Log.WithFields(logFields).Debugf("revoking AppRole assignment for %s '%s'", principalType, *r.PrincipalDisplayName)
		err := a.graphClient.ServicePrincipals().ID(a.targetId).AppRoleAssignedTo().ID(*r.ID).Request().Delete(tx.Ctx)
		if err != nil {
			return fmt.Errorf("deleting AppRole assignment for %s '%s' (%s) from '%s': %w", principalType, *r.PrincipalDisplayName, *r.PrincipalID, a.targetId, err)
		}
		tx.Log.WithFields(logFields).Infof("successfully deleted AppRole assignment for %s '%s'", principalType, *r.PrincipalDisplayName)
	}
	return nil
}

func (a appRoleAssignments) processFor(tx azure.Transaction, assignees []azure.Resource, principalType azure.PrincipalType) error {
	desired, err := a.assignFor(tx, assignees, principalType)
	if err != nil {
		return fmt.Errorf("adding app role assignments: %w", err)
	}

	existing := make([]msgraph.AppRoleAssignment, 0)
	switch principalType {
	case azure.PrincipalTypeGroup:
		existing, err = a.getAllGroups(tx.Ctx)
	case azure.PrincipalTypeServicePrincipal:
		existing, err = a.getAllServicePrincipals(tx.Ctx)
	}

	revoked := approleassignment.Difference(existing, desired)

	if err := a.revokeFor(tx, revoked, principalType); err != nil {
		return fmt.Errorf("deleting revoked app role assignments: %w", err)
	}
	return nil
}

func (a appRoleAssignments) processForGroups(tx azure.Transaction, assignees []azure.Resource) error {
	return a.processFor(tx, assignees, azure.PrincipalTypeGroup)
}

func (a appRoleAssignments) processForServicePrincipals(tx azure.Transaction, assignees []azure.Resource) error {
	return a.processFor(tx, assignees, azure.PrincipalTypeServicePrincipal)
}

func (a appRoleAssignments) toAssignment(assignee azure.ObjectId, principalType azure.PrincipalType) *msgraph.AppRoleAssignment {
	appRoleAssignment := &msgraph.AppRoleAssignment{
		AppRoleID:     &a.roleId,
		PrincipalID:   (*msgraph.UUID)(&assignee),   // Service Principal ID for the assignee, i.e. the principal that should be assigned to the app role
		ResourceID:    (*msgraph.UUID)(&a.targetId), // Service Principal ID for the target resource, i.e. the application/service principal that owns the app role
		PrincipalType: &principalType,
	}
	return appRoleAssignment
}

func assignmentInAssignments(assignment msgraph.AppRoleAssignment, assignments []msgraph.AppRoleAssignment) bool {
	for _, a := range assignments {
		equalPrincipalID := *a.PrincipalID == *assignment.PrincipalID
		equalAppRoleID := *a.AppRoleID == *assignment.AppRoleID
		equalPrincipalType := *a.PrincipalType == *assignment.PrincipalType

		if equalPrincipalID && equalAppRoleID && equalPrincipalType {
			return true
		}
	}
	return false
}

func filterByType(assignments []msgraph.AppRoleAssignment, principalType azure.PrincipalType) []msgraph.AppRoleAssignment {
	filtered := make([]msgraph.AppRoleAssignment, 0)
	for _, assignment := range assignments {
		if *assignment.PrincipalType == principalType {
			filtered = append(filtered, assignment)
		}
	}
	return filtered
}
