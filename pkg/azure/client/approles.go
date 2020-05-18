package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

const (
	DefaultAppRole   string = "access_as_application"
	DefaultAppRoleId string = "00000001-abcd-9001-0000-000000000000"
)

type appRoleAssignments struct {
	client
}

func (c client) appRoles() appRoleAssignments {
	return appRoleAssignments{c}
}

func (a appRoleAssignments) add(tx azure.Transaction, targetId azure.ServicePrincipalId, preAuthApps []azure.PreAuthorizedApp) ([]msgraphbeta.AppRoleAssignment, error) {
	assignments := make([]msgraphbeta.AppRoleAssignment, 0)
	for _, app := range preAuthApps {
		a, err := a.assign(tx, targetId, app)
		if err != nil {
			return assignments, err
		}
		if a.AppRoleID != nil && a.ResourceID != nil && a.PrincipalID != nil {
			assignments = append(assignments, a)
		}
	}
	return assignments, nil
}

func (a appRoleAssignments) exists(tx azure.Transaction, id azure.ServicePrincipalId, assignment msgraphbeta.AppRoleAssignment) (bool, error) {
	assignments, err := a.getAllFor(tx.Ctx, id)
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

func (a appRoleAssignments) update(tx azure.Transaction, targetId azure.ServicePrincipalId, preAuthApps []azure.PreAuthorizedApp) error {
	assignments, err := a.add(tx, targetId, preAuthApps)
	if err != nil {
		return fmt.Errorf("failed to add app role assignments: %w", err)
	}
	if err := a.deleteRevoked(tx, targetId, assignments); err != nil {
		return fmt.Errorf("failed to delete revoked app role assignments: %w", err)
	}
	return nil
}

func (a appRoleAssignments) assign(tx azure.Transaction, targetId azure.ServicePrincipalId, app azure.PreAuthorizedApp) (msgraphbeta.AppRoleAssignment, error) {
	spExists, assigneeSp, err := a.servicePrincipal().exists(tx.Ctx, app.ClientId)
	if err != nil {
		return msgraphbeta.AppRoleAssignment{}, err
	}
	if !spExists {
		tx.Log.Info(fmt.Sprintf("ServicePrincipal for PreAuthorizedApp (clientId '%s', name '%s') does not exist, skipping AppRole assignment...", app.ClientId, app.Name))
		return msgraphbeta.AppRoleAssignment{}, nil
	}
	assignment := a.toAssignment(targetId, *assigneeSp.ID)
	assignmentExists, err := a.exists(tx, targetId, *assignment)
	if err != nil {
		return msgraphbeta.AppRoleAssignment{}, err
	}
	if assignmentExists {
		tx.Log.Info(fmt.Sprintf("AppRole already assigned for PreAuthorizedApp (clientId '%s', name '%s'), skipping assignment...", app.ClientId, app.Name))
		return *assignment, nil
	}
	tx.Log.Info(fmt.Sprintf("AppRole not assigned for PreAuthorizedApp (clientId '%s', name '%s'), assigning...", app.ClientId, app.Name))
	_, err = a.graphBetaClient.ServicePrincipals().ID(targetId).AppRoleAssignedTo().Request().Add(tx.Ctx, assignment)
	if err != nil {
		return msgraphbeta.AppRoleAssignment{}, fmt.Errorf("failed to add AppRole assignment to target service principal ID '%s': %w", targetId, err)
	}
	tx.Log.Info(fmt.Sprintf("successfully assigned AppRole for PreAuthorizedApp (clientId '%s', name '%s')", app.ClientId, app.Name))
	return *assignment, nil
}

func (a appRoleAssignments) deleteRevoked(tx azure.Transaction, id azure.ServicePrincipalId, desiredAssignments []msgraphbeta.AppRoleAssignment) error {
	existingAssignments, err := a.getAllFor(tx.Ctx, id)
	if err != nil {
		return err
	}
	revokedAssignments := util.Difference(existingAssignments, desiredAssignments)
	for _, revoked := range revokedAssignments {
		tx.Log.Info(fmt.Sprintf("AppRole revoked for PreAuthorizedApp (servicePrincipalId '%s'), deleting assignment...", *revoked.PrincipalID))
		err = a.graphBetaClient.ServicePrincipals().ID(id).AppRoleAssignedTo().ID(*revoked.ID).Request().Delete(tx.Ctx)
		if err != nil {
			return fmt.Errorf("failed to delete revoked AppRole assignment: %w", err)
		}
		tx.Log.Info(fmt.Sprintf("successfully deleted AppRole assignment for PreAuthorizedApp (servicePrincipalId '%s')", *revoked.PrincipalID))
	}
	return nil
}

func (a appRoleAssignments) getAllFor(ctx context.Context, id azure.ServicePrincipalId) ([]msgraphbeta.AppRoleAssignment, error) {
	assignments, err := a.graphBetaClient.ServicePrincipals().ID(id).AppRoleAssignedTo().Request().GetN(ctx, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup AppRoleAssignments for service principal: %w", err)
	}
	return assignments, nil
}

func (a appRoleAssignments) toAssignment(target azure.ObjectId, assignee azure.ObjectId) *msgraphbeta.AppRoleAssignment {
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

func (a appRoleAssignments) defaultRole() msgraph.AppRole {
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
