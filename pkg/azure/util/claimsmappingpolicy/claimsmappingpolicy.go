package claimsmappingpolicy

import (
	"fmt"
	msgraph "github.com/nais/msgraph.go/v1.0"
)

type Payload struct {
	Content string `json:"@odata.id"`
}

func ToClaimsMappingPolicyPayload(policy ValidPolicy) Payload {
	return Payload{
		Content: fmt.Sprintf("https://graph.microsoft.com/v1.0/policies/claimsMappingPolicies/%s", policy.ID),
	}
}

type ClaimsMappingPolicies struct {
	Policies []ClaimsMappingPolicy `json:"value,omitempty"`
}

type ClaimsMappingPolicy struct {
	msgraph.Entity
	DisplayName *string `json:"displayName,omitempty"`
}

func (in *ClaimsMappingPolicies) Has(validPolicy ValidPolicy) bool {
	if len(in.Policies) == 0 {
		return false
	}
	for _, policy := range in.Policies {
		if *policy.ID == validPolicy.ID {
			return true
		}
	}
	return false
}
