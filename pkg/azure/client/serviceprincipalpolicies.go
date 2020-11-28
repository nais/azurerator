package client

import (
	"fmt"
	v1 "github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/claimsmappingpolicy"
)

const (
	ClaimNavIdent = "navident"
)

type servicePrincipalPolicies struct {
	servicePrincipal
}

func (s servicePrincipal) policies() servicePrincipalPolicies {
	return servicePrincipalPolicies{s}
}

func (sp servicePrincipalPolicies) assign(tx azure.Transaction, id azure.ServicePrincipalId) error {
	if tx.Instance.Spec.Claims == nil || len(tx.Instance.Spec.Claims.Extra) == 0 {
		return nil
	}

	policies := map[v1.AzureAdExtraClaim]string{
		ClaimNavIdent: sp.config.ClaimsMappingPolicy.NavIdent,
	}

	for _, claim := range tx.Instance.Spec.Claims.Extra {
		policy, found := policies[claim]
		if !found {
			continue
		}
		if err := sp.assignForPolicy(tx, policy, id); err != nil {
			return err
		}
	}
	return nil
}

func (sp servicePrincipalPolicies) assignForPolicy(tx azure.Transaction, policyID string, id azure.ServicePrincipalId) error {
	if len(policyID) == 0 {
		return nil
	}

	body := claimsmappingpolicy.ToClaimsMappingPolicyPayload(policyID)
	req := sp.graphBetaClient.ServicePrincipals().ID(id).Request()
	err := req.JSONRequest(tx.Ctx, "POST", "/claimsMappingPolicies/$ref", body, nil)

	if err != nil {
		return fmt.Errorf("assigning claims-mapping policy with ID '%s' to service principal '%s': %w", policyID, id, err)
	} else {
		tx.Log.Infof("successfully assigned claims-mapping policy with ID '%s' to service principal '%s'", policyID, id)
	}
	return nil
}
