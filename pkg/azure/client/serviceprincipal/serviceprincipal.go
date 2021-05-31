package serviceprincipal

import (
	"context"
	"fmt"

	"github.com/nais/liberator/pkg/strings"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
)

const (
	TagHideApp = "HideApp"
)

type servicePrincipal struct {
	azure.RuntimeClient
}

func NewServicePrincipal(runtimeClient azure.RuntimeClient) azure.ServicePrincipal {
	return servicePrincipal{RuntimeClient: runtimeClient}
}

func (s servicePrincipal) Owners() azure.ServicePrincipalOwners {
	return newOwners(s.RuntimeClient)
}

func (s servicePrincipal) Policies() azure.ServicePrincipalPolicies {
	return newPolicies(s.RuntimeClient)
}

func (s servicePrincipal) Register(tx azure.Transaction) (msgraph.ServicePrincipal, error) {
	clientId := tx.Instance.GetClientId()
	request := &msgraph.ServicePrincipal{
		AppID:                     &clientId,
		AppRoleAssignmentRequired: ptr.Bool(false),
		Tags:                      []string{TagHideApp},
	}
	servicePrincipal, err := s.GraphClient().ServicePrincipals().Request().Add(tx.Ctx, request)
	if err != nil {
		return msgraph.ServicePrincipal{}, fmt.Errorf("failed to register service principal: %w", err)
	}
	return *servicePrincipal, nil
}

func (s servicePrincipal) Exists(ctx context.Context, id azure.ClientId) (bool, msgraph.ServicePrincipal, error) {
	r := s.GraphClient().ServicePrincipals().Request()
	r.Filter(util.FilterByAppId(id))
	sps, err := r.GetN(ctx, s.MaxNumberOfPagesToFetch())
	if err != nil {
		return false, msgraph.ServicePrincipal{}, fmt.Errorf("failed to lookup service principal: %w", err)
	}
	if len(sps) == 0 {
		return false, msgraph.ServicePrincipal{}, nil
	}
	return true, sps[0], nil
}

func (s servicePrincipal) SetAppRoleAssignmentRequired(tx azure.Transaction) error {
	return s.setAppRoleAssignment(tx, true)
}

func (s servicePrincipal) SetAppRoleAssignmentNotRequired(tx azure.Transaction) error {
	return s.setAppRoleAssignment(tx, false)
}

func (s servicePrincipal) update(tx azure.Transaction, request *msgraph.ServicePrincipal) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	if err := s.GraphClient().ServicePrincipals().ID(servicePrincipalId).Request().Update(tx.Ctx, request); err != nil {
		return fmt.Errorf("updating service principal: %w", err)
	}
	return nil
}

func (s servicePrincipal) setAppRoleAssignment(tx azure.Transaction, required bool) error {
	exists, sp, err := s.Exists(tx.Ctx, tx.Instance.GetClientId())
	if err != nil {
		return err
	}

	if !exists || sp.AppRoleAssignmentRequired == nil {
		return fmt.Errorf("service principal not found or unexpected response data")
	}

	isAlreadySet := *sp.AppRoleAssignmentRequired == required && strings.ContainsString(sp.Tags, TagHideApp)

	if isAlreadySet {
		return nil
	}

	if required {
		tx.Log.Debug("enabling approle assignment requirement")
	} else {
		tx.Log.Debug("disabling approle assignment requirement")
	}

	request := &msgraph.ServicePrincipal{
		AppRoleAssignmentRequired: ptr.Bool(required),
		Tags:                      []string{TagHideApp},
	}

	if err := s.update(tx, request); err != nil {
		return fmt.Errorf("setting approleassignment requirement for service principal: %w", err)
	}

	return nil
}
