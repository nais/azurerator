package client

import (
	"fmt"
	v1 "github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/claimsmappingpolicy"
	"net/http"
)

const (
	ClaimNAVIdent = "NAVident"
)

type servicePrincipalPolicies struct {
	servicePrincipal
	servicePrincipalID azure.ServicePrincipalId
	assignedPolicies   *claimsmappingpolicy.ClaimsMappingPolicies
	validPolicies      map[v1.AzureAdExtraClaim]string
}

func (s servicePrincipal) policies() *servicePrincipalPolicies {
	return &servicePrincipalPolicies{servicePrincipal: s}
}

func (sp *servicePrincipalPolicies) process(tx azure.Transaction, id azure.ServicePrincipalId) error {
	if err := sp.prepare(tx, id); err != nil {
		return err
	}
	if tx.Instance.Spec.Claims == nil || len(tx.Instance.Spec.Claims.Extra) == 0 {
		return sp.revoke(tx)
	}
	return sp.assign(tx)
}

func (sp *servicePrincipalPolicies) prepare(tx azure.Transaction, id azure.ServicePrincipalId) error {
	if len(id) == 0 {
		return fmt.Errorf("service principal ID is not set")
	}
	sp.servicePrincipalID = id

	assignedPolicies, err := sp.getAssigned(tx)
	if err != nil {
		return fmt.Errorf("while preparing service principal policy assignment: %w", err)
	}
	sp.assignedPolicies = assignedPolicies

	sp.validPolicies = map[v1.AzureAdExtraClaim]string{
		ClaimNAVIdent: sp.config.ClaimsMappingPolicy.NavIdent,
	}

	return nil
}

func (sp *servicePrincipalPolicies) assign(tx azure.Transaction) error {
	for _, claim := range tx.Instance.Spec.Claims.Extra {
		policy, found := sp.validPolicies[claim]
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
	for _, policyID := range sp.validPolicies {
		if sp.assignedPolicies.Has(policyID) {
			if err := sp.removeForPolicy(tx, policyID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sp *servicePrincipalPolicies) assignForPolicy(tx azure.Transaction, policyID string) error {
	if len(policyID) == 0 {
		return nil
	}

	body := claimsmappingpolicy.ToClaimsMappingPolicyPayload(policyID)
	method := http.MethodPost
	path := "/claimsMappingPolicies/$ref"

	if err := sp.jsonRequest(tx, method, path, body, nil); err != nil {
		return fmt.Errorf("assigning claims-mapping policy '%s' to service principal '%s': %w", policyID, sp.servicePrincipalID, err)
	}

	tx.Log.Infof("successfully assigned claims-mapping policy '%s' to service principal '%s'", policyID, sp.servicePrincipalID)
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
	req := sp.graphBetaClient.ServicePrincipals().ID(sp.servicePrincipalID).Request()
	return req.JSONRequest(tx.Ctx, method, path, payload, response)
}
