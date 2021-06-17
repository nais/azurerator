package serviceprincipal

import (
	"fmt"
	"net/http"

	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/serviceprincipal/claimsmappingpolicy"
	"github.com/nais/azureator/pkg/azure/transaction"
	"github.com/nais/azureator/pkg/customresources"
)

const (
	PolicyClaimNAVIdent        = "NAVident"
	PolicyClaimAzpName         = "azp_name"
	PolicyClaimAllCustomClaims = "all"
)

type policies struct {
	azure.RuntimeClient
	servicePrincipalID azure.ServicePrincipalId
	validPolicies      *claimsmappingpolicy.ValidPolicies
}

func newPolicies(client azure.RuntimeClient) azure.ServicePrincipalPolicies {
	return &policies{
		RuntimeClient: client,
		validPolicies: &claimsmappingpolicy.ValidPolicies{
			NavIdent: claimsmappingpolicy.ValidPolicy{
				Name:     PolicyClaimNAVIdent,
				ID:       client.Config().Features.ClaimsMappingPolicies.NavIdent,
				Assigned: false,
				Desired:  false,
			},
			AzpName: claimsmappingpolicy.ValidPolicy{
				Name:     PolicyClaimAzpName,
				ID:       client.Config().Features.ClaimsMappingPolicies.AzpName,
				Assigned: false,
				Desired:  false,
			},
			AllCustomClaims: claimsmappingpolicy.ValidPolicy{
				Name:     PolicyClaimAllCustomClaims,
				ID:       client.Config().Features.ClaimsMappingPolicies.AllCustomClaims,
				Assigned: false,
				Desired:  false,
			},
		}}
}

func (p *policies) Process(tx transaction.Transaction) error {
	if err := p.prepare(tx); err != nil {
		return fmt.Errorf("preparing to process service principal policies: %w", err)
	}

	// revoke existing policies managed by this application if none found in spec
	if tx.Instance.Spec.Claims == nil || len(tx.Instance.Spec.Claims.Extra) == 0 {
		return p.revokeAllManagedPolicies(tx)
	}

	if customresources.HasExtraPolicy(tx.Instance.Spec.Claims, PolicyClaimNAVIdent) {
		p.validPolicies.NavIdent.Desired = true
	}
	if customresources.HasExtraPolicy(tx.Instance.Spec.Claims, PolicyClaimAzpName) {
		p.validPolicies.AzpName.Desired = true
	}
	if p.validPolicies.NavIdent.Desired && p.validPolicies.AzpName.Desired {
		p.validPolicies.NavIdent.Desired = false
		p.validPolicies.AzpName.Desired = false
		p.validPolicies.AllCustomClaims.Desired = true
	}

	err := p.revokeNonDesired(tx)
	if err != nil {
		return fmt.Errorf("revoking service principal policies: %w", err)
	}

	switch {
	case p.validPolicies.AllCustomClaims.Desired:
		return p.assign(tx, p.validPolicies.AllCustomClaims)
	case p.validPolicies.NavIdent.Desired:
		return p.assign(tx, p.validPolicies.NavIdent)
	case p.validPolicies.AzpName.Desired:
		return p.assign(tx, p.validPolicies.AzpName)
	}
	return nil
}

func (p *policies) prepare(tx transaction.Transaction) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()
	if len(servicePrincipalId) == 0 {
		return fmt.Errorf("service principal ID is not set")
	}
	p.servicePrincipalID = servicePrincipalId

	assignedPolicies, err := p.getAssigned(tx)
	if err != nil {
		return fmt.Errorf("fetching service principal policy assignments: %w", err)
	}

	// Graph API only allows _one_ policy assigned at a time, otherwise returns a 409 Conflict
	switch {
	case policyInPolicies(p.validPolicies.NavIdent, assignedPolicies):
		p.validPolicies.NavIdent.Assigned = true
	case policyInPolicies(p.validPolicies.AzpName, assignedPolicies):
		p.validPolicies.AzpName.Assigned = true
	case policyInPolicies(p.validPolicies.AllCustomClaims, assignedPolicies):
		p.validPolicies.AllCustomClaims.Assigned = true
	}
	return nil
}

func (p *policies) assign(tx transaction.Transaction, policy claimsmappingpolicy.ValidPolicy) error {
	if policy.Assigned {
		tx.Log.Debugf("claims-mapping policy '%s' (%s) already assigned to service principal '%s', skipping assignment", policy.Name, policy.ID, p.servicePrincipalID)
		return nil
	}

	return p.assignForPolicy(tx, policy)
}

func (p *policies) revokeNonDesired(tx transaction.Transaction) error {
	for _, validPolicy := range p.validPolicies.All() {
		if validPolicy.Assigned && !validPolicy.Desired {
			err := p.removeForPolicy(tx, validPolicy)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *policies) revokeAllManagedPolicies(tx transaction.Transaction) error {
	for _, policy := range p.validPolicies.All() {
		if policy.Assigned {
			err := p.removeForPolicy(tx, policy)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *policies) assignForPolicy(tx transaction.Transaction, policy claimsmappingpolicy.ValidPolicy) error {
	if len(policy.ID) == 0 {
		return nil
	}

	body := claimsmappingpolicy.NewPayload(policy)
	req := p.GraphClient().ServicePrincipals().ID(p.servicePrincipalID).ClaimsMappingPolicies().Request()

	err := req.JSONRequest(tx.Ctx, http.MethodPost, "/$ref", body, nil)
	if err != nil {
		return fmt.Errorf("assigning claims-mapping policy '%s' (%s) to service principal '%s': %w", policy.Name, policy.ID, p.servicePrincipalID, err)
	}

	tx.Log.Infof("successfully assigned claims-mapping policy '%s' (%s) to service principal '%s'", policy.Name, policy.ID, p.servicePrincipalID)
	return nil
}

func (p *policies) removeForPolicy(tx transaction.Transaction, policy claimsmappingpolicy.ValidPolicy) error {
	if len(policy.ID) == 0 {
		return nil
	}

	req := p.GraphClient().ServicePrincipals().ID(p.servicePrincipalID).ClaimsMappingPolicies().ID(policy.ID).Request()

	err := req.JSONRequest(tx.Ctx, http.MethodDelete, "/$ref", nil, nil)
	if err != nil {
		return fmt.Errorf("removing claims-mapping policy '%s' (%s) from service principal '%s'", policy.Name, policy.ID, p.servicePrincipalID)
	}

	tx.Log.Infof("successfully removed claims-mapping policy '%s' (%s) from service principal '%s'", policy.Name, policy.ID, p.servicePrincipalID)
	return nil
}

func (p *policies) getAssigned(tx transaction.Transaction) ([]msgraph.ClaimsMappingPolicy, error) {
	response, err := p.GraphClient().ServicePrincipals().ID(p.servicePrincipalID).ClaimsMappingPolicies().Request().Get(tx.Ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching claims-mapping policies for service principal '%s': %w", p.servicePrincipalID, err)
	}

	return response, nil
}

func policyInPolicies(validPolicy claimsmappingpolicy.ValidPolicy, policies []msgraph.ClaimsMappingPolicy) bool {
	if len(policies) == 0 {
		return false
	}

	for _, policy := range policies {
		if *policy.ID == validPolicy.ID {
			return true
		}
	}
	return false
}
