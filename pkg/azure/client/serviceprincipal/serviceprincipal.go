package serviceprincipal

import (
	"context"
	"fmt"

	cache "github.com/Code-Hex/go-generics-cache"
	"github.com/nais/liberator/pkg/strings"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/transaction"
)

const (
	TagHideApp = "HideApp"
)

var clientIdCache = cache.New[azure.ServicePrincipalId, azure.ClientId]()
var servicePrincipalIdCache = cache.New[azure.ClientId, azure.ServicePrincipalId]()

type ServicePrincipal interface {
	Owners() Owners
	Policies() Policies

	GetClientId(ctx context.Context, id azure.ServicePrincipalId) (azure.ClientId, error)
	GetIdByClientId(ctx context.Context, id azure.ClientId) (azure.ServicePrincipalId, error)
	Exists(ctx context.Context, id azure.ClientId) (bool, msgraph.ServicePrincipal, error)
	Register(tx transaction.Transaction) (msgraph.ServicePrincipal, error)
	SetAppRoleAssignmentRequired(tx transaction.Transaction) error
	SetAppRoleAssignmentNotRequired(tx transaction.Transaction) error
}

type servicePrincipal struct {
	azure.RuntimeClient
}

func NewServicePrincipal(runtimeClient azure.RuntimeClient) ServicePrincipal {
	return servicePrincipal{RuntimeClient: runtimeClient}
}

func (s servicePrincipal) Owners() Owners {
	return newOwners(s.RuntimeClient)
}

func (s servicePrincipal) Policies() Policies {
	return newPolicies(s.RuntimeClient)
}

func (s servicePrincipal) GetClientId(ctx context.Context, id azure.ServicePrincipalId) (azure.ClientId, error) {
	if val, found := clientIdCache.Get(id); found {
		return val, nil
	}

	sp, err := s.GraphClient().ServicePrincipals().ID(id).Request().Get(ctx)
	if err != nil {
		return "", fmt.Errorf("fetching service principal with id '%s': %w", id, err)
	}

	clientId := *sp.AppID
	clientIdCache.Set(id, clientId)

	return clientId, nil
}

func (s servicePrincipal) GetIdByClientId(ctx context.Context, id azure.ClientId) (azure.ServicePrincipalId, error) {
	if val, found := servicePrincipalIdCache.Get(id); found {
		return val, nil
	}

	exists, sp, err := s.Exists(ctx, id)
	if err != nil {
		return "", err
	}

	if !exists {
		return "", fmt.Errorf("service principal with client id '%s' not found", id)
	}

	servicePrincipalId := *sp.ID
	servicePrincipalIdCache.Set(id, servicePrincipalId)

	return servicePrincipalId, nil
}

func (s servicePrincipal) Register(tx transaction.Transaction) (msgraph.ServicePrincipal, error) {
	clientId := tx.Instance.GetClientId()
	request := &msgraph.ServicePrincipal{
		AppID:                     &clientId,
		AppRoleAssignmentRequired: ptr.Bool(s.Config().Features.AppRoleAssignmentRequired.Enabled),
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

	sp := sps[0]
	clientIdCache.Set(*sp.ID, *sp.AppID)

	return true, sp, nil
}

func (s servicePrincipal) SetAppRoleAssignmentRequired(tx transaction.Transaction) error {
	return s.setAppRoleAssignment(tx, true)
}

func (s servicePrincipal) SetAppRoleAssignmentNotRequired(tx transaction.Transaction) error {
	return s.setAppRoleAssignment(tx, false)
}

func (s servicePrincipal) update(tx transaction.Transaction, request *msgraph.ServicePrincipal) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	if err := s.GraphClient().ServicePrincipals().ID(servicePrincipalId).Request().Update(tx.Ctx, request); err != nil {
		return fmt.Errorf("updating service principal: %w", err)
	}
	return nil
}

func (s servicePrincipal) setAppRoleAssignment(tx transaction.Transaction, required bool) error {
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
		tx.Logger.Debug("enabling approle assignment requirement")
	} else {
		tx.Logger.Debug("disabling approle assignment requirement")
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
