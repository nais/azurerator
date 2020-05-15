package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

const (
	DefaultAppRole   string = "access_as_application"
	DefaultAppRoleId string = "00000001-abcd-9001-0000-000000000000"
)

func (c client) addAppRoleAssignments(tx azure.Transaction, targetId azure.ServicePrincipalId, preAuthApps []azure.PreAuthorizedApp) ([]msgraphbeta.AppRoleAssignment, error) {
	assignments := make([]msgraphbeta.AppRoleAssignment, 0)
	for _, app := range preAuthApps {
		assignment, err := c.assignAppRole(tx, targetId, app)
		if err != nil {
			return assignments, err
		}
		if assignment.AppRoleID != nil && assignment.ResourceID != nil && assignment.PrincipalID != nil {
			assignments = append(assignments, assignment)
		}
	}
	return assignments, nil
}

func (c client) updateAppRoles(tx azure.Transaction, targetId azure.ServicePrincipalId, preAuthApps []azure.PreAuthorizedApp) error {
	assignments, err := c.addAppRoleAssignments(tx, targetId, preAuthApps)
	if err != nil {
		return fmt.Errorf("failed to add app role assignments: %w", err)
	}
	if err := c.deleteRevokedAppRoleAssignments(tx, targetId, assignments); err != nil {
		return fmt.Errorf("failed to delete revoked app role assignments: %w", err)
	}
	return nil
}

func (c client) appRoleAssignmentExists(tx azure.Transaction, id azure.ServicePrincipalId, assignment msgraphbeta.AppRoleAssignment) (bool, error) {
	assignments, err := c.getAllAppRoleAssignmentsFor(tx.Ctx, id)
	if err != nil {
		return false, err
	}
	for _, a := range assignments {
		if *a.PrincipalID == *assignment.PrincipalID {
			return true, nil
		}
	}
	return false, nil
}

func (c client) assignAppRole(tx azure.Transaction, targetId azure.ServicePrincipalId, app azure.PreAuthorizedApp) (msgraphbeta.AppRoleAssignment, error) {
	spExists, assigneeSp, err := c.servicePrincipalExists(tx.Ctx, app.ClientId)
	if err != nil {
		return msgraphbeta.AppRoleAssignment{}, err
	}
	if !spExists {
		tx.Log.Info(fmt.Sprintf("ServicePrincipal for PreAuthorizedApp (clientId '%s', name '%s') does not exist, skipping AppRole assignment...", app.ClientId, app.Name))
		return msgraphbeta.AppRoleAssignment{}, nil
	}
	assignment := toAppRoleAssignment(targetId, *assigneeSp.ID)
	assignmentExists, err := c.appRoleAssignmentExists(tx, targetId, *assignment)
	if err != nil {
		return msgraphbeta.AppRoleAssignment{}, err
	}
	if assignmentExists {
		tx.Log.Info(fmt.Sprintf("AppRole already assigned for PreAuthorizedApp (clientId '%s', name '%s'), skipping assignment...", app.ClientId, app.Name))
		return *assignment, nil
	}
	tx.Log.Info(fmt.Sprintf("AppRole not assigned for PreAuthorizedApp (clientId '%s', name '%s'), assigning...", app.ClientId, app.Name))
	_, err = c.graphBetaClient.ServicePrincipals().ID(targetId).AppRoleAssignedTo().Request().Add(tx.Ctx, assignment)
	if err != nil {
		return msgraphbeta.AppRoleAssignment{}, fmt.Errorf("failed to add AppRole assignment to target service principal ID '%s': %w", targetId, err)
	}
	tx.Log.Info(fmt.Sprintf("successfully assigned AppRole for PreAuthorizedApp (clientId '%s', name '%s')", app.ClientId, app.Name))
	return *assignment, nil
}

func (c client) deleteRevokedAppRoleAssignments(tx azure.Transaction, id azure.ServicePrincipalId, desiredAssignments []msgraphbeta.AppRoleAssignment) error {
	existingAssignments, err := c.getAllAppRoleAssignmentsFor(tx.Ctx, id)
	if err != nil {
		return err
	}
	revokedAssignments := difference(existingAssignments, desiredAssignments)
	for _, revoked := range revokedAssignments {
		tx.Log.Info(fmt.Sprintf("AppRole revoked for PreAuthorizedApp (servicePrincipalId '%s'), deleting assignment...", *revoked.PrincipalID))
		err = c.graphBetaClient.ServicePrincipals().ID(id).AppRoleAssignedTo().ID(*revoked.ID).Request().Delete(tx.Ctx)
		if err != nil {
			return fmt.Errorf("failed to delete revoked AppRole assignment: %w", err)
		}
		tx.Log.Info(fmt.Sprintf("successfully deleted AppRole assignment for PreAuthorizedApp (servicePrincipalId '%s')", *revoked.PrincipalID))
	}
	return nil
}

func (c client) getAllAppRoleAssignmentsFor(ctx context.Context, id azure.ServicePrincipalId) ([]msgraphbeta.AppRoleAssignment, error) {
	assignments, err := c.graphBetaClient.ServicePrincipals().ID(id).AppRoleAssignedTo().Request().GetN(ctx, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup AppRoleAssignments for service principal: %w", err)
	}
	return assignments, nil
}

func toAppRoleAssignment(target azure.ObjectId, assignee azure.ObjectId) *msgraphbeta.AppRoleAssignment {
	appRoleId := msgraphbeta.UUID(DefaultAppRoleId)
	principalId := msgraphbeta.UUID(assignee) // Service Principal ID for the assignee, i.e. the application that should be assigned to the app role
	resourceId := msgraphbeta.UUID(target)    // Service Principal ID for the target resource, i.e. the application that owns the app role
	appRoleAssignment := &msgraphbeta.AppRoleAssignment{
		AppRoleID:   &appRoleId,
		PrincipalID: &principalId,
		ResourceID:  &resourceId,
	}
	return appRoleAssignment
}

func defaultAppRole() msgraph.AppRole {
	roleId := msgraph.UUID(DefaultAppRoleId)
	allowedMemberTypes := []string{"Application"}
	return msgraph.AppRole{
		Object:             msgraph.Object{},
		AllowedMemberTypes: allowedMemberTypes,
		Description:        ptr.String(DefaultAppRole),
		DisplayName:        ptr.String(DefaultAppRole),
		ID:                 &roleId,
		IsEnabled:          ptr.Bool(true),
		Value:              ptr.String(DefaultAppRole),
	}
}

// difference returns the elements in `a` that aren't in `b`
// shamelessly stolen and modified from https://stackoverflow.com/a/45428032/11868133
func difference(a []msgraphbeta.AppRoleAssignment, b []msgraphbeta.AppRoleAssignment) []msgraphbeta.AppRoleAssignment {
	mb := make(map[msgraphbeta.UUID]struct{}, len(b))
	for _, x := range b {
		mb[*x.PrincipalID] = struct{}{}
	}
	diff := make([]msgraphbeta.AppRoleAssignment, 0)
	for _, x := range a {
		if _, found := mb[*x.PrincipalID]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
