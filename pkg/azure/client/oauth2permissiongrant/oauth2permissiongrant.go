package oauth2permissiongrant

import (
	"fmt"

	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/transaction"
)

type OAuth2PermissionGrant interface {
	Process(tx transaction.Transaction) error
}

type oAuth2PermissionGrant struct {
	azure.RuntimeClient
}

func NewOAuth2PermissionGrant(runtimeClient azure.RuntimeClient) OAuth2PermissionGrant {
	return oAuth2PermissionGrant{RuntimeClient: runtimeClient}
}

func (o oAuth2PermissionGrant) Process(tx transaction.Transaction) error {
	exists, err := o.exists(tx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	_, err = o.GraphClient().OAuth2PermissionGrants().Request().Add(tx.Ctx, o.toGrant(servicePrincipalId))
	if err != nil {
		return fmt.Errorf("registering oauth2 permission grants: %w", err)
	}
	return nil
}

func (o oAuth2PermissionGrant) exists(tx transaction.Transaction) (bool, error) {
	// For some odd reason Graph has defined 'clientId' in the oAuth2PermissionGrant resource to be the _objectId_
	// for the ServicePrincipal when referring to the id of the ServicePrincipal granted consent...
	clientId := tx.Instance.GetServicePrincipalId()
	r := o.GraphClient().OAuth2PermissionGrants().Request()
	r.Filter(util.FilterByClientId(clientId))
	grants, err := r.GetN(tx.Ctx, o.MaxNumberOfPagesToFetch())
	if err != nil {
		return false, fmt.Errorf("looking up oauth2 permission grants: %w", err)
	}
	return len(grants) > 0, nil
}

// OAuth2 permission grants allows us to pre-approve this application for the defined scopes/permissions set.
// This results in the enduser not having to manually consent whenever interacting with the application, e.g. during
// an OIDC login flow.
func (o oAuth2PermissionGrant) toGrant(servicePrincipalId azure.ServicePrincipalId) *msgraph.OAuth2PermissionGrant {
	return &msgraph.OAuth2PermissionGrant{
		ClientID:    ptr.String(servicePrincipalId),
		ConsentType: ptr.String("AllPrincipals"),
		ResourceID:  ptr.String(o.Config().PermissionGrantResourceId),
		Scope:       ptr.String("openid User.Read GroupMember.Read.All"),
	}
}
