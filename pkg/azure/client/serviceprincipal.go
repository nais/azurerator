package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/ptr"
)

const (
	ServicePrincipalTagHideApp = "HideApp"
)

type servicePrincipal struct {
	client
}

func (c client) servicePrincipal() servicePrincipal {
	return servicePrincipal{c}
}

func (s servicePrincipal) register(tx azure.Transaction) (msgraphbeta.ServicePrincipal, error) {
	clientId := tx.Instance.GetClientId()
	request := &msgraphbeta.ServicePrincipal{
		AppID:                     &clientId,
		AppRoleAssignmentRequired: ptr.Bool(false),
		Tags:                      []string{ServicePrincipalTagHideApp},
	}
	servicePrincipal, err := s.graphBetaClient.ServicePrincipals().Request().Add(tx.Ctx, request)
	if err != nil {
		return msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to register service principal: %w", err)
	}
	return *servicePrincipal, nil
}

func (s servicePrincipal) exists(ctx context.Context, id azure.ClientId) (bool, msgraphbeta.ServicePrincipal, error) {
	r := s.graphBetaClient.ServicePrincipals().Request()
	r.Filter(util.FilterByAppId(id))
	sps, err := r.GetN(ctx, MaxNumberOfPagesToFetch)
	if err != nil {
		return false, msgraphbeta.ServicePrincipal{}, fmt.Errorf("failed to lookup service principal: %w", err)
	}
	if len(sps) == 0 {
		return false, msgraphbeta.ServicePrincipal{}, nil
	}
	return true, sps[0], nil
}

func (s servicePrincipal) update(tx azure.Transaction, request *msgraphbeta.ServicePrincipal) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	if err := s.graphBetaClient.ServicePrincipals().ID(servicePrincipalId).Request().Update(tx.Ctx, request); err != nil {
		return fmt.Errorf("updating service principal: %w", err)
	}
	return nil
}

func (s servicePrincipal) setAppRoleAssignmentRequired(tx azure.Transaction) error {
	tx.Log.Debug("enabling approle assignment requirement")
	return s.setAppRoleAssignment(tx, true)
}

func (s servicePrincipal) setAppRoleAssignmentNotRequired(tx azure.Transaction) error {
	tx.Log.Debug("disabling approle assignment requirement")
	return s.setAppRoleAssignment(tx, false)
}

func (s servicePrincipal) setAppRoleAssignment(tx azure.Transaction, required bool) error {
	request := &msgraphbeta.ServicePrincipal{
		AppRoleAssignmentRequired: ptr.Bool(required),
		Tags:                      []string{ServicePrincipalTagHideApp},
	}

	if err := s.update(tx, request); err != nil {
		return fmt.Errorf("setting approleassignment requirement for service principal: %w", err)
	}
	return nil
}
