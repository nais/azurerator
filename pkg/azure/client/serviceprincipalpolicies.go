package client

import (
	"fmt"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/claimsmappingpolicy"
	"github.com/nais/azureator/pkg/customresources"
	"net/http"
)

const (
	ClaimNAVIdent = "NAVident"
)

type servicePrincipalPolicies struct {
	servicePrincipal
	servicePrincipalID azure.ServicePrincipalId
	assignedPolicies   *claimsmappingpolicy.ClaimsMappingPolicies
	validPolicies      claimsmappingpolicy.ValidPolicies
}

func (s servicePrincipal) policies() *servicePrincipalPolicies {
	return &servicePrincipalPolicies{
		servicePrincipal: s,
		validPolicies: claimsmappingpolicy.ValidPolicies{
			Policies: []claimsmappingpolicy.ValidPolicy{
				{
					Name: ClaimNAVIdent,
					ID:   s.config.Features.ClaimsMappingPolicies.NavIdent,
				},
			},
		},
	}
}

func (sp *servicePrincipalPolicies) process(tx azure.Transaction) error {
	if err := sp.prepare(tx); err != nil {
		return fmt.Errorf("preparing to process service principal policies: %w", err)
	}

	if tx.Instance.Spec.Claims == nil || len(tx.Instance.Spec.Claims.Extra) == 0 {
		// revoke existing policies managed by this application if not found in spec
		return sp.revokeAllManagedPolicies(tx)
	}

	if err := sp.revoke(tx); err != nil {
		return fmt.Errorf("revoking service principal policies: %w", err)
	}

	if err := sp.assign(tx); err != nil {
		return fmt.Errorf("assigning service principal policies: %w", err)
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
		return fmt.Errorf("while preparing service principal policy assignment: %w", err)
	}
	sp.assignedPolicies = assignedPolicies

	return nil
}

func (sp *servicePrincipalPolicies) assign(tx azure.Transaction) error {
	for _, claim := range tx.Instance.Spec.Claims.Extra {
		policy, found := sp.validPolicies.HasPolicyByName(claim)
		if !found {
			continue
		}

		if sp.assignedPolicies.Has(policy) {
			tx.Log.Debugf("claims-mapping policy '%s' already assigned to service principal '%s', skipping assignment", policy, sp.servicePrincipalID)
			continue
		}

		if err := sp.assignForPolicy(tx, policy); err != nil {
			return err
		}
	}
	return nil
}

func (sp *servicePrincipalPolicies) revoke(tx azure.Transaction) error {
	for _, assignedPolicy := range sp.assignedPolicies.Policies {
		validPolicy, found := sp.validPolicies.HasPolicyByID(*assignedPolicy.ID)
		if !found || customresources.HasExtraPolicy(tx.Instance.Spec.Claims, validPolicy.Name) {
			continue
		}

		if err := sp.removeForPolicy(tx, *assignedPolicy.ID); err != nil {
			return err
		}
	}
	return nil
}

func (sp *servicePrincipalPolicies) revokeAllManagedPolicies(tx azure.Transaction) error {
	for _, policy := range sp.validPolicies.Policies {
		if sp.assignedPolicies.Has(policy) {
			if err := sp.removeForPolicy(tx, policy.ID); err != nil {
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
	method := http.MethodPost
	path := "/claimsMappingPolicies/$ref"

	if err := sp.jsonRequest(tx, method, path, body, nil); err != nil {
		return fmt.Errorf("assigning claims-mapping policy '%s' to service principal '%s': %w", policy.ID, sp.servicePrincipalID, err)
	}

	tx.Log.Infof("successfully assigned claims-mapping policy '%s' to service principal '%s'", policy.ID, sp.servicePrincipalID)
	return nil
}

func (sp *servicePrincipalPolicies) removeForPolicy(tx azure.Transaction, policyID string) error {
	if len(policyID) == 0 {
		return nil
	}

	method := http.MethodDelete
	path := fmt.Sprintf("/claimsMappingPolicies/%s/$ref", policyID)

	if err := sp.jsonRequest(tx, method, path, nil, nil); err != nil {
		return fmt.Errorf("removing claims-mapping policy '%s' from service principal '%s'", policyID, sp.servicePrincipalID)
	}

	tx.Log.Infof("successfully removed claims-mapping policy '%s' from service principal '%s'", policyID, sp.servicePrincipalID)
	return nil
}

func (sp *servicePrincipalPolicies) getAssigned(tx azure.Transaction) (*claimsmappingpolicy.ClaimsMappingPolicies, error) {
	method := http.MethodGet
	path := "/claimsMappingPolicies"
	response := &claimsmappingpolicy.ClaimsMappingPolicies{}

	if err := sp.jsonRequest(tx, method, path, nil, response); err != nil {
		return nil, fmt.Errorf("fetching claims-mapping policies for service principal '%s': %w", sp.servicePrincipalID, err)
	}

	return response, nil
}

func (sp *servicePrincipalPolicies) jsonRequest(tx azure.Transaction, method, path string, payload, response interface{}) error {
	req := sp.graphClient.ServicePrincipals().ID(sp.servicePrincipalID).Request()
	return req.JSONRequest(tx.Ctx, method, path, payload, response)
}
