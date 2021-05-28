package client

import (
	"context"
	"fmt"

	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/approleassignment"
)

type appRoleAssignments struct {
	client
	targetId  azure.ObjectId
	logFields log.Fields
}

type appRoleAssignmentsWithRoleId struct {
	appRoleAssignments
	roleId msgraph.UUID
}

func (c client) appRoleAssignments(roleId msgraph.UUID, targetId azure.ObjectId) appRoleAssignmentsWithRoleId {
	a := appRoleAssignmentsWithRoleId{
		appRoleAssignments: c.appRoleAssignmentsNoRoleId(targetId),
		roleId:             roleId,
	}
	a.appRoleAssignments.logFields["roleId"] = roleId
	return a
}

func (c client) appRoleAssignmentsNoRoleId(targetId azure.ObjectId) appRoleAssignments {
	return appRoleAssignments{
		client:   c,
		targetId: targetId,
		logFields: log.Fields{
			"targetId": targetId,
		},
	}
}

func (a appRoleAssignments) request() *msgraph.ServicePrincipalAppRoleAssignedToCollectionRequest {
	return a.graphClient.ServicePrincipals().ID(a.targetId).AppRoleAssignedTo().Request()
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
	groups := approleassignment.FilterByType(assignments, azure.PrincipalTypeGroup)
	return groups, nil
}

func (a appRoleAssignments) getAllServicePrincipals(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	assignments, err := a.getAll(ctx)
	if err != nil {
		return nil, err
	}
	servicePrincipals := approleassignment.FilterByType(assignments, azure.PrincipalTypeServicePrincipal)
	return servicePrincipals, nil
}

func (a appRoleAssignmentsWithRoleId) getAll(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	assignments, err := a.appRoleAssignments.getAll(ctx)
	if err != nil {
		return nil, err
	}
	return approleassignment.FilterByRoleID(assignments, a.roleId), nil
}

func (a appRoleAssignmentsWithRoleId) getAllGroups(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	groups, err := a.appRoleAssignments.getAllGroups(ctx)
	if err != nil {
		return nil, err
	}
	return approleassignment.FilterByRoleID(groups, a.roleId), nil
}

func (a appRoleAssignmentsWithRoleId) getAllServicePrincipals(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	servicePrincipals, err := a.appRoleAssignments.getAllServicePrincipals(ctx)
	if err != nil {
		return nil, err
	}
	return approleassignment.FilterByRoleID(servicePrincipals, a.roleId), nil
}

func (a appRoleAssignmentsWithRoleId) revokeFor(tx azure.Transaction, revoked []msgraph.AppRoleAssignment, principalType azure.PrincipalType) error {
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

func (a appRoleAssignmentsWithRoleId) assignFor(tx azure.Transaction, assignees []azure.Resource, principalType azure.PrincipalType) ([]msgraph.AppRoleAssignment, error) {
	assignments := make([]msgraph.AppRoleAssignment, 0)

	existing, err := a.getAll(tx.Ctx)
	if err != nil {
		return nil, fmt.Errorf("looking up existing AppRole assignments: %w", err)
	}

	for _, assignee := range assignees {
		logFields := a.logFields
		logFields["assigneeClientId"] = assignee.ClientId
		logFields["assigneeObjectId"] = assignee.ObjectId
		logFields["assigneePrincipalType"] = assignee.PrincipalType

		if len(assignee.ObjectId) == 0 {
			tx.Log.WithFields(logFields).Debugf("skipping AppRole assignment: object ID for %s '%s' is not set", assignee.PrincipalType, assignee.Name)
			continue
		}

		if principalType != assignee.PrincipalType {
			tx.Log.WithFields(logFields).Debugf("skipping AppRole assignment: principal type %s (object ID '%s') does not match expected principal type %s", assignee.PrincipalType, assignee.ObjectId, principalType)
			continue
		}

		assignment, err := a.toAssignment(assignee)
		if err != nil {
			return nil, err
		}

		if approleassignment.AssignmentInAssignments(*assignment, existing) {
			tx.Log.WithFields(logFields).Infof("skipping AppRole assignment: already assigned for %s '%s'", assignee.PrincipalType, assignee.Name)
			assignments = append(assignments, *assignment)
			continue
		}

		tx.Log.WithFields(logFields).Debugf("assigning AppRole for %s '%s'...", assignee.PrincipalType, assignee.Name)
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

func (a appRoleAssignmentsWithRoleId) processFor(tx azure.Transaction, assignees []azure.Resource, principalType azure.PrincipalType) error {
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

func (a appRoleAssignmentsWithRoleId) processForGroups(tx azure.Transaction, assignees []azure.Resource) error {
	return a.processFor(tx, assignees, azure.PrincipalTypeGroup)
}

func (a appRoleAssignmentsWithRoleId) processForServicePrincipals(tx azure.Transaction, assignees []azure.Resource) error {
	return a.processFor(tx, assignees, azure.PrincipalTypeServicePrincipal)
}

func (a appRoleAssignmentsWithRoleId) toAssignment(assignee azure.Resource) (*msgraph.AppRoleAssignment, error) {
	return approleassignment.ToAssignment(a.roleId, assignee.ObjectId, a.targetId, assignee.PrincipalType)
}
