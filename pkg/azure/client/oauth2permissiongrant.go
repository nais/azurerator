package client

import (
	"fmt"
	"time"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
)

type oAuth2PermissionGrant struct {
	client
}

func (c client) oAuth2PermissionGrant() oAuth2PermissionGrant {
	return oAuth2PermissionGrant{c}
}

func (o oAuth2PermissionGrant) exists(tx azure.Transaction) (bool, error) {
	// For some odd reason Graph has defined 'clientId' in the oAuth2PermissionGrant resource to be the _objectId_
	// for the ServicePrincipal when referring to the id of the ServicePrincipal granted consent...
	clientId := tx.Instance.GetServicePrincipalId()
	r := o.graphBetaClient.OAuth2PermissionGrants().Request()
	r.Filter(util.FilterByClientId(clientId))
	grants, err := r.GetN(tx.Ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return false, fmt.Errorf("looking up oauth2 permission grants: %w", err)
	}
	return len(grants) > 0, nil
}

func (o oAuth2PermissionGrant) process(tx azure.Transaction) error {
	exists, err := o.exists(tx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	_, err = o.graphBetaClient.OAuth2PermissionGrants().Request().Add(tx.Ctx, o.toGrant(servicePrincipalId, o.config.PermissionGrantResourceId))
	if err != nil {
		return fmt.Errorf("registering oauth2 permission grants: %w", err)
	}
	return nil
}

// OAuth2 permission grants allows us to pre-approve this application for the defined scopes/permissions set.
// This results in the enduser not having to manually consent whenever interacting with the application, e.g. during
// an OIDC login flow.
func (o oAuth2PermissionGrant) toGrant(servicePrincipalId azure.ServicePrincipalId, permissionGrantResourceId string) *msgraphbeta.OAuth2PermissionGrant {
	// This field is required by Graph API, but isn't actually used as per 2020-04-29.
	// https://docs.microsoft.com/en-us/graph/api/resources/oauth2permissiongrant?view=graph-rest-beta
	expiryTime := time.Date(0001, time.January, 1, 0, 0, 0, 0, time.UTC)
	return &msgraphbeta.OAuth2PermissionGrant{
		ClientID:    ptr.String(servicePrincipalId),
		ConsentType: ptr.String("AllPrincipals"),
		ExpiryTime:  &expiryTime,
		ResourceID:  ptr.String(permissionGrantResourceId),
		Scope:       ptr.String("openid User.Read GroupMember.Read.All"),
	}
}
