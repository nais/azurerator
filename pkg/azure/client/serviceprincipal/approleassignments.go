package serviceprincipal

import (
	"fmt"

	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/approleassignment"
	"github.com/nais/azureator/pkg/azure/permissions"
	"github.com/nais/azureator/pkg/azure/resource"
	"github.com/nais/azureator/pkg/azure/transaction"
)

type operation string

const (
	operationSkipped  = operation("skipped (already assigned)")
	operationRevoked  = operation("revoked")
	operationAssigned = operation("assigned")
)

const unknownRole = "UNKNOWN_ROLE"

type AppRoleAssignments interface {
	GetAll() (approleassignment.List, error)
	GetAllGroups() (approleassignment.List, error)
	GetAllServicePrincipals() (approleassignment.List, error)
	ProcessForGroups(assignees resource.Resources, roles permissions.Permissions) error
	ProcessForServicePrincipals(assignees resource.Resources, roles permissions.Permissions) error
}

type appRoleAssignments struct {
	azure.RuntimeClient
	tx        transaction.Transaction
	targetId  azure.ServicePrincipalId
	logFields log.Fields
}

func NewAppRoleAssignments(client azure.RuntimeClient, tx transaction.Transaction, targetId azure.ServicePrincipalId) AppRoleAssignments {
	return appRoleAssignments{
		RuntimeClient: client,
		tx:            tx,
		targetId:      targetId,
		logFields: log.Fields{
			"targetId": targetId,
		},
	}
}

func (a appRoleAssignments) GetAll() (approleassignment.List, error) {
	assignments, err := a.request().GetN(a.tx.Ctx, a.MaxNumberOfPagesToFetch())
	if err != nil {
		return nil, fmt.Errorf("looking up AppRole assignments for service principal '%s': %w", a.targetId, err)
	}

	return assignments, nil
}

func (a appRoleAssignments) GetAllGroups() (approleassignment.List, error) {
	assignments, err := a.GetAll()
	if err != nil {
		return nil, err
	}

	return assignments.Groups(), nil
}

func (a appRoleAssignments) GetAllServicePrincipals() (approleassignment.List, error) {
	assignments, err := a.GetAll()
	if err != nil {
		return nil, err
	}

	return assignments.ServicePrincipals(), nil
}

func (a appRoleAssignments) ProcessForGroups(assignees resource.Resources, roles permissions.Permissions) error {
	return a.processFor(assignees, resource.PrincipalTypeGroup, roles)
}

func (a appRoleAssignments) ProcessForServicePrincipals(assignees resource.Resources, roles permissions.Permissions) error {
	return a.processFor(assignees, resource.PrincipalTypeServicePrincipal, roles)
}

func (a appRoleAssignments) processFor(assignees resource.Resources, principalType resource.PrincipalType, roles permissions.Permissions) error {
	// only fetch existing assignments for a given principal type
	existingAssignments, err := a.fetchExisting(principalType)
	if err != nil {
		return fmt.Errorf("looking up existing AppRole assignments: %w", err)
	}

	// ensure that we only process assignees of the given principal type
	assignees = assignees.FilterByPrincipalType(principalType)

	err = a.processEnabledRoles(existingAssignments, roles, assignees, principalType)
	if err != nil {
		return err
	}

	err = a.revokeAssignmentsForDisabledRoles(existingAssignments, roles)
	if err != nil {
		return err
	}

	err = a.revokeAssignmentsWithoutMatchingDesiredRole(existingAssignments, roles)
	if err != nil {
		return err
	}

	return nil
}

func (a appRoleAssignments) fetchExisting(principalType resource.PrincipalType) (approleassignment.List, error) {
	switch principalType {
	case resource.PrincipalTypeGroup:
		return a.GetAllGroups()
	case resource.PrincipalTypeServicePrincipal:
		return a.GetAllServicePrincipals()
	default:
		return nil, fmt.Errorf("'%s' is not a supported principal type", principalType)
	}
}

func (a appRoleAssignments) processEnabledRoles(existing approleassignment.List, roles permissions.Permissions, assignees resource.Resources, principalType resource.PrincipalType) error {
	for _, role := range roles.Enabled() {
		existingByRole := existing.FilterByRoleID(role.ID)
		desiredAssignees := assignees.ExtractDesiredAssignees(principalType, role)

		desired := approleassignment.ToAppRoleAssignments(desiredAssignees, a.targetId, role)
		toAssign := approleassignment.ToAssign(existingByRole, desired)
		toRevoke := approleassignment.ToRevoke(existingByRole, desired)
		unmodified := approleassignment.Unmodified(existingByRole, toAssign, toRevoke)

		err := a.assignFor(toAssign, role.Name)
		if err != nil {
			return fmt.Errorf("assigning desired approleassignments for role '%s' (%s): %w", role.Name, role.ID, err)
		}

		err = a.revokeFor(toRevoke, role.Name)
		if err != nil {
			return fmt.Errorf("revoking non-desired approleassignments for role '%s' (%s): %w", role.Name, role.ID, err)
		}

		a.logUnmodified(unmodified, role.Name)
	}

	return nil
}

func (a appRoleAssignments) revokeAssignmentsForDisabledRoles(existing approleassignment.List, roles permissions.Permissions) error {
	for _, role := range roles.Disabled() {
		a.tx.Log.WithFields(a.logFields).Debugf("revoking assignments for disabled AppRole '%s' (%s)...", role.Name, role.ID)

		err := a.revokeFor(existing.FilterByRoleID(role.ID), role.Name)
		if err != nil {
			return fmt.Errorf("revoking assignments for non-desired/disabled role '%s' (%s): %w", role.Name, role.ID, err)
		}
	}

	return nil
}

func (a appRoleAssignments) revokeAssignmentsWithoutMatchingDesiredRole(existing approleassignment.List, roles permissions.Permissions) error {
	toRevoke := existing.WithoutMatchingRole(roles)

	if len(toRevoke) > 0 {
		a.tx.Log.WithFields(a.logFields).Debugf("revoking assignments for non-existing AppRoles...")

		err := a.revokeFor(toRevoke, unknownRole)
		if err != nil {
			return fmt.Errorf("revoking assignments for non-existing roles: %w", err)
		}
	}

	return nil
}

func (a appRoleAssignments) assignFor(toAssign approleassignment.List, roleName string) error {
	for _, assignment := range toAssign {
		err := a.logAndDo(assignment, operationAssigned, roleName, func() error {
			_, err := a.request().Add(a.tx.Ctx, &assignment)
			return err
		})

		if err != nil {
			return err
		}
	}
	return nil
}

func (a appRoleAssignments) revokeFor(revoked approleassignment.List, roleName string) error {
	for _, assignment := range revoked {
		err := a.logAndDo(assignment, operationRevoked, roleName, func() error {
			return a.requestWithID(*assignment.ID).Delete(a.tx.Ctx)
		})

		if err != nil {
			return err
		}
	}
	return nil
}

func (a appRoleAssignments) logUnmodified(unmodified approleassignment.List, roleName string) {
	for _, assignment := range unmodified {
		_ = a.logAndDo(assignment, operationSkipped, roleName, func() error { return nil })
	}
}

func (a appRoleAssignments) logAndDo(assignment msgraph.AppRoleAssignment, operation operation, roleName string, do func() error) error {
	assigneeName := *assignment.PrincipalDisplayName
	assigneeObjectId := *assignment.PrincipalID
	assigneePrincipalType := *assignment.PrincipalType
	roleId := *assignment.AppRoleID

	logFields := a.logFields
	logFields["assigneeName"] = assigneeName
	logFields["assigneeObjectId"] = assigneeObjectId
	logFields["assigneePrincipalType"] = assigneePrincipalType
	logFields["roleId"] = roleId
	logFields["roleName"] = roleName

	if err := do(); err != nil {
		return fmt.Errorf(
			"processing AppRole assignment for %s '%s' (%s) with role '%s' (%s) and target service principal ID '%s': %w",
			assigneePrincipalType, assigneeName, assigneeObjectId, roleName, roleId, a.targetId, err,
		)
	}

	a.tx.Log.WithFields(logFields).
		Infof(
			"%s AppRole assignment for %s '%s' to role '%s'.",
			operation, assigneePrincipalType, assigneeName, roleName,
		)

	return nil
}

func (a appRoleAssignments) request() *msgraph.ServicePrincipalAppRoleAssignedToCollectionRequest {
	return a.GraphClient().ServicePrincipals().ID(a.targetId).AppRoleAssignedTo().Request()
}

func (a appRoleAssignments) requestWithID(resourceID string) *msgraph.AppRoleAssignmentRequest {
	return a.GraphClient().ServicePrincipals().ID(a.targetId).AppRoleAssignedTo().ID(resourceID).Request()
}
