package approleassignment

import (
	"context"
	"fmt"

	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/approleassignment"
)

// TODO(tronghn): should log AppRole ID / name being processed
//  should also refactor Resource to contain list of all (AppRole ID / name) instead of locking the instance here to a single role
type appRoleAssignmentsWithRoleId struct {
	azure.RuntimeClient
	azure.AppRoleAssignments
	RoleId msgraph.UUID
}

func NewAppRoleAssignmentsWithRoleId(client azure.RuntimeClient, appRoleAssignments azure.AppRoleAssignments, roleId msgraph.UUID) azure.AppRoleAssignmentsWithRoleId {
	a := appRoleAssignmentsWithRoleId{
		RuntimeClient:      client,
		AppRoleAssignments: appRoleAssignments,
		RoleId:             roleId,
	}
	a.AppRoleAssignments.LogFields()["roleId"] = roleId
	return a
}

func (a appRoleAssignmentsWithRoleId) ProcessForGroups(tx azure.Transaction, assignees []azure.Resource) error {
	return a.processFor(tx, assignees, azure.PrincipalTypeGroup)
}

func (a appRoleAssignmentsWithRoleId) ProcessForServicePrincipals(tx azure.Transaction, assignees []azure.Resource) error {
	return a.processFor(tx, assignees, azure.PrincipalTypeServicePrincipal)
}

// TODO(tronghn): extract getAll() from assignFor() and reuse in assignFor / getAllGroups / getAllServicePrincipals?
//  difference = existing (before new assignments) + newAssigned) - desired (newAssigned + alreadyAssigned
//  difference = existing (before new assignments) - alreadyAssigned?
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

// TODO(tronghn): should return struct desiredAssignmentsResult of ([]newAssigned, []alreadyAssigned, []invalid)
func (a appRoleAssignmentsWithRoleId) assignFor(tx azure.Transaction, assignees []azure.Resource, principalType azure.PrincipalType) ([]msgraph.AppRoleAssignment, error) {
	assignments := make([]msgraph.AppRoleAssignment, 0)

	existing, err := a.getAll(tx.Ctx)
	if err != nil {
		return nil, fmt.Errorf("looking up existing AppRole assignments: %w", err)
	}

	for _, assignee := range assignees {
		logFields := a.LogFields()
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

		if assignmentInAssignments(*assignment, existing) {
			tx.Log.WithFields(logFields).Infof("skipping AppRole assignment: already assigned for %s '%s'", assignee.PrincipalType, assignee.Name)
			assignments = append(assignments, *assignment)
			continue
		}

		tx.Log.WithFields(logFields).Debugf("assigning AppRole for %s '%s'...", assignee.PrincipalType, assignee.Name)
		result, err := a.Request().Add(tx.Ctx, assignment)
		if err != nil {
			return nil, fmt.Errorf("assigning AppRole for %s '%s' (%s) to target service principal ID '%s': %w", assignee.PrincipalType, assignee.Name, assignee.ObjectId, a.TargetId(), err)
		}

		if result.AppRoleID != nil && result.ResourceID != nil && result.PrincipalID != nil {
			tx.Log.WithFields(logFields).Infof("successfully assigned AppRole for %s '%s'", assignee.PrincipalType, assignee.Name)
			assignments = append(assignments, *result)
		}
	}
	return assignments, nil
}

func (a appRoleAssignmentsWithRoleId) revokeFor(tx azure.Transaction, revoked []msgraph.AppRoleAssignment, principalType azure.PrincipalType) error {
	for _, r := range revoked {
		logFields := a.LogFields()
		logFields["assigneeObjectId"] = *r.PrincipalID

		tx.Log.WithFields(logFields).Debugf("revoking AppRole assignment for %s '%s'", principalType, *r.PrincipalDisplayName)
		err := a.GraphClient().ServicePrincipals().ID(a.TargetId()).AppRoleAssignedTo().ID(*r.ID).Request().Delete(tx.Ctx)
		if err != nil {
			return fmt.Errorf("deleting AppRole assignment for %s '%s' (%s) from '%s': %w", principalType, *r.PrincipalDisplayName, *r.PrincipalID, a.TargetId(), err)
		}
		tx.Log.WithFields(logFields).Infof("successfully deleted AppRole assignment for %s '%s'", principalType, *r.PrincipalDisplayName)
	}
	return nil
}

func (a appRoleAssignmentsWithRoleId) getAll(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	assignments, err := a.AppRoleAssignments.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return filterByRoleID(assignments, a.RoleId), nil
}

func (a appRoleAssignmentsWithRoleId) getAllGroups(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	groups, err := a.AppRoleAssignments.GetAllGroups(ctx)
	if err != nil {
		return nil, err
	}
	return filterByRoleID(groups, a.RoleId), nil
}

func (a appRoleAssignmentsWithRoleId) getAllServicePrincipals(ctx context.Context) ([]msgraph.AppRoleAssignment, error) {
	servicePrincipals, err := a.AppRoleAssignments.GetAllServicePrincipals(ctx)
	if err != nil {
		return nil, err
	}
	return filterByRoleID(servicePrincipals, a.RoleId), nil
}

func (a appRoleAssignmentsWithRoleId) toAssignment(assignee azure.Resource) (*msgraph.AppRoleAssignment, error) {
	return &msgraph.AppRoleAssignment{
		AppRoleID:     &a.RoleId,                                 // The ID of the AppRole belonging to the target resource to be assigned
		PrincipalID:   (*msgraph.UUID)(&assignee.ObjectId),       // Service Principal ID for the assignee, i.e. the principal that should be assigned to the app role
		ResourceID:    (*msgraph.UUID)(ptr.String(a.TargetId())), // Service Principal ID for the target resource, i.e. the application/service principal that owns the app role
		PrincipalType: (*string)(&assignee.PrincipalType),
	}, nil
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

func filterByRoleID(assignments []msgraph.AppRoleAssignment, roleId msgraph.UUID) []msgraph.AppRoleAssignment {
	filtered := make([]msgraph.AppRoleAssignment, 0)
	for _, assignment := range assignments {
		if *assignment.AppRoleID == roleId {
			filtered = append(filtered, assignment)
		}
	}
	return filtered
}
