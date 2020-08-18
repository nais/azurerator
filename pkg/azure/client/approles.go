package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/approle"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

const (
	DefaultAppRole     string = "access_as_application"
	DefaultAppRoleId   string = "00000001-abcd-9001-0000-000000000000"
	PrincipalTypeGroup string = "Group"
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
		tx.Log.Debugf("skipping AppRole assignment: ServicePrincipal for PreAuthorizedApp (clientId '%s', name '%s') does not exist", app.ClientId, app.Name)
		return msgraphbeta.AppRoleAssignment{}, nil
	}
	assignment := a.toAssignment(targetId, *assigneeSp.ID)
	assignmentExists, err := a.exists(tx, targetId, *assignment)
	if err != nil {
		return msgraphbeta.AppRoleAssignment{}, err
	}
	if assignmentExists {
		tx.Log.Infof("skipping AppRole assignment: already assigned for PreAuthorizedApp (clientId '%s', name '%s')", app.ClientId, app.Name)
		return *assignment, nil
	}
	tx.Log.Debugf("assigning AppRole for PreAuthorizedApp (clientId '%s', name '%s')...", app.ClientId, app.Name)
	_, err = a.request(targetId).Add(tx.Ctx, assignment)
	if err != nil {
		return msgraphbeta.AppRoleAssignment{}, fmt.Errorf("failed to add AppRole assignment to target service principal ID '%s': %w", targetId, err)
	}
	tx.Log.Infof("successfully assigned AppRole for PreAuthorizedApp (clientId '%s', name '%s')", app.ClientId, app.Name)
	return *assignment, nil
}

func (a appRoleAssignments) getRevoked(tx azure.Transaction, id azure.ServicePrincipalId, desired []msgraphbeta.AppRoleAssignment) ([]msgraphbeta.AppRoleAssignment, error) {
	existing, err := a.getAllFor(tx.Ctx, id)
	if err != nil {
		return nil, err
	}
	revoked := approle.Difference(existing, desired)
	return revoked, nil
}

func (a appRoleAssignments) deleteRevoked(tx azure.Transaction, id azure.ServicePrincipalId, desired []msgraphbeta.AppRoleAssignment) error {
	revoked, err := a.getRevoked(tx, id, desired)
	if err != nil {
		return err
	}
	for _, r := range revoked {
		tx.Log.Debugf("deleting AppRole assignment: revoked for PreAuthorizedApp (servicePrincipalId '%s')", *r.PrincipalID)
		if err := a.delete(tx, id, r); err != nil {
			return err
		}
	}
	return nil
}

func (a appRoleAssignments) delete(tx azure.Transaction, id azure.ServicePrincipalId, revoked msgraphbeta.AppRoleAssignment) error {
	err := a.graphBetaClient.ServicePrincipals().ID(id).AppRoleAssignedTo().ID(*revoked.ID).Request().Delete(tx.Ctx)
	if err != nil {
		return fmt.Errorf("failed to delete revoked AppRole assignment: %w", err)
	}
	tx.Log.Infof("successfully deleted AppRole assignment for PreAuthorizedApp (servicePrincipalId '%s')", *revoked.PrincipalID)
	return nil
}

func (a appRoleAssignments) getAllFor(ctx context.Context, id azure.ServicePrincipalId) ([]msgraphbeta.AppRoleAssignment, error) {
	assignments, err := a.request(id).GetN(ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup AppRoleAssignments for service principal: %w", err)
	}
	return assignments, nil
}

func (a appRoleAssignments) getAssignedGroups(ctx context.Context, id azure.ServicePrincipalId) ([]msgraphbeta.AppRoleAssignment, error) {
	assignments, err := a.getAllFor(ctx, id)
	if err != nil {
		return nil, err
	}
	groups := make([]msgraphbeta.AppRoleAssignment, 0)
	for _, assignment := range assignments {
		principalType := *assignment.PrincipalType
		if principalType == PrincipalTypeGroup {
			groups = append(groups, assignment)
		}
	}
	return groups, nil
}

func (a appRoleAssignments) request(id azure.ServicePrincipalId) *msgraphbeta.ServicePrincipalAppRoleAssignedToCollectionRequest {
	return a.graphBetaClient.ServicePrincipals().ID(id).AppRoleAssignedTo().Request()
}

func (a appRoleAssignments) toAssignment(target azure.ObjectId, assignee azure.ObjectId) *msgraphbeta.AppRoleAssignment {
	appRoleId := msgraphbeta.UUID(DefaultAppRoleId)
	appRoleAssignment := &msgraphbeta.AppRoleAssignment{
		AppRoleID:   &appRoleId,
		PrincipalID: (*msgraphbeta.UUID)(&assignee), // Service Principal ID for the assignee, i.e. the application that should be assigned to the app role
		ResourceID:  (*msgraphbeta.UUID)(&target),   // Service Principal ID for the target resource, i.e. the application that owns the app role
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
