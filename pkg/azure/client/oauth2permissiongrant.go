package client

import (
	"context"
	"fmt"
	"time"

	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
)

func (c client) registerOAuth2PermissionGrants(ctx context.Context, principal msgraphbeta.ServicePrincipal) error {
	_, err := c.graphBetaClient.Oauth2PermissionGrants().Request().Add(ctx, toOAuth2PermissionGrants(&principal, c.config.PermissionGrantResourceId))
	if err != nil {
		return fmt.Errorf("failed to register oauth2 permission grants: %w", err)
	}
	return nil
}

func (c client) oAuth2PermissionGrantsExist(ctx context.Context, sp msgraphbeta.ServicePrincipal) (bool, error) {
	clientId := *sp.ID
	r := c.graphBetaClient.Oauth2PermissionGrants().Request()
	r.Filter(util.FilterByClientId(clientId))
	grants, err := r.GetN(ctx, 1000)
	if err != nil {
		return false, fmt.Errorf("failed to lookup oauth2 permission grants: %w", err)
	}
	return len(grants) > 0, nil
}

func (c client) upsertOAuth2PermissionGrants(ctx context.Context, sp msgraphbeta.ServicePrincipal) error {
	exists, err := c.oAuth2PermissionGrantsExist(ctx, sp)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if err := c.registerOAuth2PermissionGrants(ctx, sp); err != nil {
		return err
	}
	return nil
}

// OAuth2 permission grants allows us to pre-approve this application for the defined scopes/permissions set.
// This results in the enduser not having to manually consent whenever interacting with the application, e.g. during
// an OIDC login flow.
func toOAuth2PermissionGrants(servicePrincipal *msgraphbeta.ServicePrincipal, permissionGrantResourceId string) *msgraphbeta.OAuth2PermissionGrant {
	// This field is required by Graph API, but isn't actually used as per 2020-04-29.
	// https://docs.microsoft.com/en-us/graph/api/resources/oauth2permissiongrant?view=graph-rest-beta
	expiryTime := time.Date(0001, time.January, 1, 0, 0, 0, 0, time.UTC)
	return &msgraphbeta.OAuth2PermissionGrant{
		ClientID:    ptr.String(*servicePrincipal.ID),
		ConsentType: ptr.String("AllPrincipals"),
		ExpiryTime:  &expiryTime,
		ResourceID:  ptr.String(permissionGrantResourceId),
		Scope:       ptr.String("openid User.Read"),
	}
}
