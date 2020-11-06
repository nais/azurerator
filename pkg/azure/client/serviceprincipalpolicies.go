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

func (sp servicePrincipalPolicies) assign(tx azure.Transaction) error {
	if len(tx.Instance.Spec.AdditionalClaims) == 0 {
		return nil
	}

	policies := map[v1.AzureAdAdditionalClaim]string{
		ClaimNavIdent: sp.config.ClaimsMappingPolicy.NavIdent,
	}

	for _, claim := range tx.Instance.Spec.AdditionalClaims {
		policy, found := policies[claim]
		if !found {
			continue
		}
		if err := sp.assignForPolicy(tx, policy); err != nil {
			return err
		}
	}
	return nil
}

func (sp servicePrincipalPolicies) assignForPolicy(tx azure.Transaction, policyID string) error {
	if len(policyID) == 0 {
		return nil
	}

	servicePrincipalId := tx.Instance.Status.ServicePrincipalId

	body := claimsmappingpolicy.ToClaimsMappingPolicyPayload(policyID)
	req := sp.graphBetaClient.ServicePrincipals().ID(servicePrincipalId).Request()
	err := req.JSONRequest(tx.Ctx, "POST", "/claimsMappingPolicies/$ref", body, nil)

	if err != nil {
		tx.Log.Error(fmt.Errorf("assigning claims-mapping policy with ID '%s' to service principal '%s': %w", policyID, servicePrincipalId, err))
	}
	tx.Log.Info("successfully assigned claims-mapping policy with ID '%s' to service principal '%s'", policyID, servicePrincipalId)
	return nil
}
