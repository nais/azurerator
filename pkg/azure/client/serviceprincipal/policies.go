package serviceprincipal

import (
	"context"
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

	assignedPolicies, err := p.getAssignedPolicies(tx.Ctx, servicePrincipalID)
	if err != nil {
		return fmt.Errorf("claims-mapping-policies: fetching existing policies for service principal '%s': %w", servicePrincipalID, err)
	}

	// a ServicePrincipal can only have one assignedPolicy assigned at any given time, so we must first revoke any existing, non-matching policies
	for _, assignedPolicy := range assignedPolicies {
		assignedPolicyID, ok := policyID(assignedPolicy)
		if !ok {
			continue
		}

		// return early if the desired policy is already assigned
		if assignedPolicyID == desiredPolicyID {
			tx.Logger.Debugf("claims-mapping-policies: skipping assignment; '%s' already assigned to service principal '%s'", desiredPolicyID, servicePrincipalID)
			return nil
		}

		err := p.removePolicy(tx.Ctx, assignedPolicyID, servicePrincipalID)
		if err != nil {
			return fmt.Errorf("claims-mapping-policies: removing '%s' from service principal '%s': %w", assignedPolicyID, servicePrincipalID, err)
		}
		tx.Logger.Infof("claims-mapping-policies: successfully removed '%s' from service principal '%s'", assignedPolicyID, servicePrincipalID)
	}

	err = p.assignPolicy(tx.Ctx, desiredPolicyID, servicePrincipalID)
	if err != nil {
		return fmt.Errorf("claims-mapping-policies: assigning '%s' to service principal '%s': %w", desiredPolicyID, servicePrincipalID, err)
	}
	tx.Logger.Infof("claims-mapping-policies: successfully assigned '%s' to service principal '%s'", desiredPolicyID, servicePrincipalID)
	return nil
}

func (p *policies) assignPolicy(ctx context.Context, desiredPolicyID, servicePrincipalID string) error {
	return p.GraphClient().
		ServicePrincipals().
		ID(servicePrincipalID).
		ClaimsMappingPolicies().
		Request().
		JSONRequest(ctx, http.MethodPost, "/$ref", NewClaimsMappingPolicyBody(desiredPolicyID), nil)
}

func (p *policies) getAssignedPolicies(ctx context.Context, servicePrincipalID string) ([]msgraph.ClaimsMappingPolicy, error) {
	return p.GraphClient().
		ServicePrincipals().
		ID(servicePrincipalID).
		ClaimsMappingPolicies().
		Request().
		Get(ctx)
}

func (p *policies) removePolicy(ctx context.Context, assignedPolicyID, servicePrincipalID string) error {
	return p.GraphClient().
		ServicePrincipals().
		ID(servicePrincipalID).
		ClaimsMappingPolicies().
		ID(assignedPolicyID).
		Request().
		JSONRequest(ctx, http.MethodDelete, "/$ref", nil, nil)
}

func policyID(policy msgraph.ClaimsMappingPolicy) (string, bool) {
	if policy.ID != nil && *policy.ID != "" {
		return *policy.ID, true
	}

	return "", false
}
