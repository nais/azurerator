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

func (c client) addAppRoleAssignments(tx azure.Transaction, sp msgraphbeta.ServicePrincipal) error {
	for _, app := range tx.Resource.Spec.PreAuthorizedApplications {
		exists, err := c.preAuthAppExists(tx.Ctx, app)
		if err != nil {
			return fmt.Errorf("failed to lookup existence of pre-authorized app (clientId '%s', name '%s'): %w", app.ClientId, app.Name, err)
		}
		if !exists {
			tx.Log.Info(fmt.Sprintf("pre-authorized app (clientId '%s', name '%s') does not exist, skipping approle assignment...", app.ClientId, app.Name))
			continue
		}
		clientId, err := c.getClientId(tx.Ctx, app)
		if err != nil {
			return err
		}
		otherSpId, err := c.getServicePrincipalId(tx.Ctx, clientId)
		if err != nil {
			return err
		}
		assignment := toAppRoleAssignment(*sp.ID, otherSpId)
		if err := c.assignAppRole(tx.Ctx, sp, assignment); err != nil {
			return fmt.Errorf("failed to add approle assignment to service principal: %w", err)
		}
	}
	return nil
}

func (c client) assignAppRole(ctx context.Context, sp msgraphbeta.ServicePrincipal, assignment msgraphbeta.AppRoleAssignment) error {
	_, err := c.graphBetaClient.ServicePrincipals().ID(*sp.ID).AppRoleAssignments().Request().Add(ctx, &assignment)
	if err != nil {
		return fmt.Errorf("failed to update service principal with id '%s': %w", *sp.ID, err)
	}
	return nil
}

// todo - remove approles for applications removed from preauthorizedapps
func (c client) deleteRevokedAppRoleAssignments(tx azure.Transaction, principal msgraphbeta.ServicePrincipal) error {
	return nil
}

func toAppRoleAssignment(selfSpId string, otherSpId string) msgraphbeta.AppRoleAssignment {
	appRoleId := msgraphbeta.UUID(DefaultAppRoleId)
	principalId := msgraphbeta.UUID(otherSpId) // Service Principal ID for application that should be assigned to the app role
	resourceId := msgraphbeta.UUID(selfSpId)   // Service Principal ID for the application that owns the app role
	appRoleAssignment := msgraphbeta.AppRoleAssignment{
		AppRoleID:   &appRoleId,
		PrincipalID: &principalId,
		ResourceID:  &resourceId,
	}
	return appRoleAssignment
}

func toApprole(roleName string) msgraph.AppRole {
	roleId := msgraph.UUID(DefaultAppRoleId)
	allowedmembertypes := []string{"Application"}
	return msgraph.AppRole{
		Object:             msgraph.Object{},
		AllowedMemberTypes: allowedmembertypes,
		Description:        ptr.String(roleName),
		DisplayName:        ptr.String(roleName),
		ID:                 &roleId,
		IsEnabled:          ptr.Bool(true),
		Value:              ptr.String(roleName),
	}
}
