package client

import (
	"fmt"
	"net/http"

	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/claimsmappingpolicy"
	"github.com/nais/azureator/pkg/customresources"
)

const (
	PolicyClaimNAVIdent        = "NAVident"
	PolicyClaimAzpName         = "azp_name"
	PolicyClaimAllCustomClaims = "all"
)

type servicePrincipalPolicies struct {
	servicePrincipal
	servicePrincipalID azure.ServicePrincipalId
	validPolicies      *claimsmappingpolicy.ValidPolicies
}

func (s servicePrincipal) policies() *servicePrincipalPolicies {
	return &servicePrincipalPolicies{
		servicePrincipal: s,
		validPolicies: &claimsmappingpolicy.ValidPolicies{
			NavIdent: claimsmappingpolicy.ValidPolicy{
				Name:     PolicyClaimNAVIdent,
				ID:       s.config.Features.ClaimsMappingPolicies.NavIdent,
				Assigned: false,
				Desired:  false,
			},
			AzpName: claimsmappingpolicy.ValidPolicy{
				Name:     PolicyClaimAzpName,
				ID:       s.config.Features.ClaimsMappingPolicies.AzpName,
				Assigned: false,
				Desired:  false,
			},
			AllCustomClaims: claimsmappingpolicy.ValidPolicy{
				Name:     PolicyClaimAllCustomClaims,
				ID:       s.config.Features.ClaimsMappingPolicies.AllCustomClaims,
				Assigned: false,
				Desired:  false,
			},
		},
	}
}

func (sp *servicePrincipalPolicies) process(tx azure.Transaction) error {
	if err := sp.prepare(tx); err != nil {
		return fmt.Errorf("preparing to process service principal policies: %w", err)
	}

	// revoke existing policies managed by this application if none found in spec
	if tx.Instance.Spec.Claims == nil || len(tx.Instance.Spec.Claims.Extra) == 0 {
		return sp.revokeAllManagedPolicies(tx)
	}

	if customresources.HasExtraPolicy(tx.Instance.Spec.Claims, PolicyClaimNAVIdent) {
		sp.validPolicies.NavIdent.Desired = true
	}
	if customresources.HasExtraPolicy(tx.Instance.Spec.Claims, PolicyClaimAzpName) {
		sp.validPolicies.AzpName.Desired = true
	}
	if sp.validPolicies.NavIdent.Desired && sp.validPolicies.AzpName.Desired {
		sp.validPolicies.NavIdent.Desired = false
		sp.validPolicies.AzpName.Desired = false
		sp.validPolicies.AllCustomClaims.Desired = true
	}

	err := sp.revokeNonDesired(tx)
	if err != nil {
		return fmt.Errorf("revoking service principal policies: %w", err)
	}

	switch {
	case sp.validPolicies.AllCustomClaims.Desired:
		return sp.assign(tx, sp.validPolicies.AllCustomClaims)
	case sp.validPolicies.NavIdent.Desired:
		return sp.assign(tx, sp.validPolicies.NavIdent)
	case sp.validPolicies.AzpName.Desired:
		return sp.assign(tx, sp.validPolicies.AzpName)
	}
	return nil
}

func (sp *servicePrincipalPolicies) prepare(tx azure.Transaction) error {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()
	if len(servicePrincipalId) == 0 {
		return fmt.Errorf("service principal ID is not set")
	}
	sp.servicePrincipalID = servicePrincipalId

	assignedPolicies, err := sp.getAssigned(tx)
	if err != nil {
		return fmt.Errorf("fetching service principal policy assignments: %w", err)
	}

	// Graph API only allows _one_ policy assigned at a time, otherwise returns a 409 Conflict
	switch {
	case claimsmappingpolicy.PolicyInPolicies(sp.validPolicies.NavIdent, assignedPolicies):
		sp.validPolicies.NavIdent.Assigned = true
	case claimsmappingpolicy.PolicyInPolicies(sp.validPolicies.AzpName, assignedPolicies):
		sp.validPolicies.AzpName.Assigned = true
	case claimsmappingpolicy.PolicyInPolicies(sp.validPolicies.AllCustomClaims, assignedPolicies):
		sp.validPolicies.AllCustomClaims.Assigned = true
	}
	return nil
}

func (sp *servicePrincipalPolicies) assign(tx azure.Transaction, policy claimsmappingpolicy.ValidPolicy) error {
	if policy.Assigned {
		tx.Log.Debugf("claims-mapping policy '%s' (%s) already assigned to service principal '%s', skipping assignment", policy.Name, policy.ID, sp.servicePrincipalID)
		return nil
	}

	return sp.assignForPolicy(tx, policy)
}

func (sp *servicePrincipalPolicies) revokeNonDesired(tx azure.Transaction) error {
	for _, validPolicy := range sp.validPolicies.All() {
		if validPolicy.Assigned && !validPolicy.Desired {
			err := sp.removeForPolicy(tx, validPolicy)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (sp *servicePrincipalPolicies) revokeAllManagedPolicies(tx azure.Transaction) error {
	for _, policy := range sp.validPolicies.All() {
		if policy.Assigned {
			err := sp.removeForPolicy(tx, policy)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (sp *servicePrincipalPolicies) assignForPolicy(tx azure.Transaction, policy claimsmappingpolicy.ValidPolicy) error {
	if len(policy.ID) == 0 {
		return nil
	}

	body := claimsmappingpolicy.ToClaimsMappingPolicyPayload(policy)
	req := sp.graphClient.ServicePrincipals().ID(sp.servicePrincipalID).ClaimsMappingPolicies().Request()

	err := req.JSONRequest(tx.Ctx, http.MethodPost, "/$ref", body, nil)
	if err != nil {
		return fmt.Errorf("assigning claims-mapping policy '%s' (%s) to service principal '%s': %w", policy.Name, policy.ID, sp.servicePrincipalID, err)
	}

	tx.Log.Infof("successfully assigned claims-mapping policy '%s' (%s) to service principal '%s'", policy.Name, policy.ID, sp.servicePrincipalID)
	return nil
}

func (sp *servicePrincipalPolicies) removeForPolicy(tx azure.Transaction, policy claimsmappingpolicy.ValidPolicy) error {
	if len(policy.ID) == 0 {
		return nil
	}

	req := sp.graphClient.ServicePrincipals().ID(sp.servicePrincipalID).ClaimsMappingPolicies().ID(policy.ID).Request()

	err := req.JSONRequest(tx.Ctx, http.MethodDelete, "/$ref", nil, nil)
	if err != nil {
		return fmt.Errorf("removing claims-mapping policy '%s' (%s) from service principal '%s'", policy.Name, policy.ID, sp.servicePrincipalID)
	}

	tx.Log.Infof("successfully removed claims-mapping policy '%s' (%s) from service principal '%s'", policy.Name, policy.ID, sp.servicePrincipalID)
	return nil
}

func (sp *servicePrincipalPolicies) getAssigned(tx azure.Transaction) ([]msgraph.ClaimsMappingPolicy, error) {
	response, err := sp.graphClient.ServicePrincipals().ID(sp.servicePrincipalID).ClaimsMappingPolicies().Request().Get(tx.Ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching claims-mapping policies for service principal '%s': %w", sp.servicePrincipalID, err)
	}

	return response, nil
}
