package serviceprincipal

import (
	"fmt"
	"net/http"

	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/transaction"
)

type Policies interface {
	Process(tx transaction.Transaction, policyID string) error
}

type policies struct {
	azure.RuntimeClient
}

type ClaimsMappingPolicyBody struct {
	Content string `json:"@odata.id"`
}

func NewClaimsMappingPolicyBody(id string) ClaimsMappingPolicyBody {
	return ClaimsMappingPolicyBody{
		Content: fmt.Sprintf("https://graph.microsoft.com/v1.0/policies/claimsMappingPolicies/%s", id),
	}
}

func newPolicies(client azure.RuntimeClient) Policies {
	return &policies{
		RuntimeClient: client,
	}
}

func (p *policies) Process(tx transaction.Transaction, desiredPolicyID string) error {
	if desiredPolicyID == "" {
		tx.Logger.Debug("claims-mapping-policies: desiredPolicyID is empty; skipping...")
		return nil
	}

	servicePrincipalID := tx.Instance.GetServicePrincipalId()
	if len(servicePrincipalID) == 0 {
		return fmt.Errorf("claims-mapping-policies: service principal ID is not set")
	}

	assignedPolicies, err := p.getAssignedPolicies(tx, servicePrincipalID)
	if err != nil {
		return fmt.Errorf("claims-mapping-policies: fetching existing policies for service principal '%s': %w", servicePrincipalID, err)
	}

	if hasPolicyAssignment(assignedPolicies, desiredPolicyID) {
		tx.Logger.Debugf("claims-mapping-policies: skipping assignment; '%s' already assigned to service principal '%s'", desiredPolicyID, servicePrincipalID)
		return nil
	}

	// a ServicePrincipal can only have one assignedPolicy assigned at any given time, so we must first revoke any existing, non-matching policies
	for _, assignedPolicy := range assignedPolicies {
		assignedPolicyID, ok := policyID(assignedPolicy)
		if !ok {
			continue
		}

		err := p.removePolicy(tx, assignedPolicyID, servicePrincipalID)
		if err != nil {
			return fmt.Errorf("claims-mapping-policies: removing '%s' from service principal '%s': %w", assignedPolicyID, servicePrincipalID, err)
		}

		tx.Logger.Infof("claims-mapping-policies: successfully removed '%s' from service principal '%s'", assignedPolicyID, servicePrincipalID)
	}

	err = p.assignPolicy(tx, desiredPolicyID, servicePrincipalID)
	if err != nil {
		return fmt.Errorf("claims-mapping-policies: assigning '%s' to service principal '%s': %w", desiredPolicyID, servicePrincipalID, err)
	}

	tx.Logger.Infof("claims-mapping-policies: successfully assigned '%s' to service principal '%s'", desiredPolicyID, servicePrincipalID)
	return nil
}

func (p *policies) assignPolicy(tx transaction.Transaction, desiredPolicyID, servicePrincipalID string) error {
	return p.GraphClient().
		ServicePrincipals().
		ID(servicePrincipalID).
		ClaimsMappingPolicies().
		Request().
		JSONRequest(tx.Ctx, http.MethodPost, "/$ref", NewClaimsMappingPolicyBody(desiredPolicyID), nil)
}

func (p *policies) getAssignedPolicies(tx transaction.Transaction, servicePrincipalID string) ([]msgraph.ClaimsMappingPolicy, error) {
	return p.GraphClient().
		ServicePrincipals().
		ID(servicePrincipalID).
		ClaimsMappingPolicies().
		Request().
		Get(tx.Ctx)
}

func (p *policies) removePolicy(tx transaction.Transaction, assignedPolicyID, servicePrincipalID string) error {
	return p.GraphClient().
		ServicePrincipals().
		ID(servicePrincipalID).
		ClaimsMappingPolicies().
		ID(assignedPolicyID).
		Request().
		JSONRequest(tx.Ctx, http.MethodDelete, "/$ref", nil, nil)
}

func hasPolicyAssignment(policies []msgraph.ClaimsMappingPolicy, id string) bool {
	for _, policy := range policies {
		if policy.ID != nil && *policy.ID == id {
			return true
		}
	}

	return false
}

func policyID(policy msgraph.ClaimsMappingPolicy) (string, bool) {
	if policy.ID == nil || *policy.ID == "" {
		return "", false
	}

	return *policy.ID, true
}
