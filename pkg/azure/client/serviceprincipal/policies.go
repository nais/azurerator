package serviceprincipal

import (
	"fmt"
	"net/http"

	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/transaction"
)

type Policies interface {
	Process(tx transaction.Transaction) error
}

type policies struct {
	azure.RuntimeClient
	assignedPolicies   []msgraph.ClaimsMappingPolicy
	servicePrincipalID azure.ServicePrincipalId
	policyID           string
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
		policyID:      client.Config().Features.ClaimsMappingPolicies.ID,
	}
}

func (p *policies) Process(tx transaction.Transaction) error {
	if err := p.prepare(tx); err != nil {
		return fmt.Errorf("preparing to process service principal policies: %w", err)
	}

	// revoke existing policies if no custom claims found in spec
	if tx.Instance.Spec.Claims == nil || len(tx.Instance.Spec.Claims.Extra) == 0 {
		return p.revokeExistingPolicies(tx)
	}

	return p.assign(tx, p.policyID)
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

	p.assignedPolicies = assignedPolicies
	return nil
}

func (p *policies) hasPolicyAssignment(id string) bool {
	for _, policy := range p.assignedPolicies {
		if policy.ID != nil && *policy.ID == id {
			return true
		}
	}

	return false
}

func (p *policies) assign(tx transaction.Transaction, id string) error {
	if p.hasPolicyAssignment(p.policyID) {
		tx.Logger.Debugf("skipping claims-mapping policy assignment; '%s' already assigned to service principal '%s'", p.policyID, p.servicePrincipalID)
		return nil
	}

	// a ServicePrincipal can only have one policy assigned at any given time, so we must first revoke any existing, non-matching policies
	err := p.revokeExistingPolicies(tx)
	if err != nil {
		return fmt.Errorf("revoking non-matching policies: %w", err)
	}

	return p.assignForPolicy(tx, id)
}

func (p *policies) revokeExistingPolicies(tx transaction.Transaction) error {
	for _, policy := range p.assignedPolicies {
		if policy.ID == nil {
			continue
		}

		err := p.removeForPolicy(tx, *policy.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *policies) assignForPolicy(tx transaction.Transaction, id string) error {
	if len(id) == 0 {
		return nil
	}

	body := NewClaimsMappingPolicyBody(id)
	req := p.GraphClient().ServicePrincipals().ID(p.servicePrincipalID).ClaimsMappingPolicies().Request()

	err := req.JSONRequest(tx.Ctx, http.MethodPost, "/$ref", body, nil)
	if err != nil {
		return fmt.Errorf("assigning claims-mapping policy '%s' to service principal '%s': %w", id, p.servicePrincipalID, err)
	}

	tx.Logger.Infof("successfully assigned claims-mapping policy '%s' to service principal '%s'", id, p.servicePrincipalID)
	return nil
}

func (p *policies) removeForPolicy(tx transaction.Transaction, id string) error {
	if len(id) == 0 {
		return nil
	}

	req := p.GraphClient().ServicePrincipals().ID(p.servicePrincipalID).ClaimsMappingPolicies().ID(id).Request()

	err := req.JSONRequest(tx.Ctx, http.MethodDelete, "/$ref", nil, nil)
	if err != nil {
		return fmt.Errorf("removing claims-mapping policy '%s' from service principal '%s'", id, p.servicePrincipalID)
	}

	tx.Logger.Infof("successfully removed claims-mapping policy '%s' from service principal '%s'", id, p.servicePrincipalID)
	return nil
}

func (p *policies) getAssigned(tx transaction.Transaction) ([]msgraph.ClaimsMappingPolicy, error) {
	response, err := p.GraphClient().ServicePrincipals().ID(p.servicePrincipalID).ClaimsMappingPolicies().Request().Get(tx.Ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching claims-mapping policies for service principal '%s': %w", p.servicePrincipalID, err)
	}

	return response, nil
}
