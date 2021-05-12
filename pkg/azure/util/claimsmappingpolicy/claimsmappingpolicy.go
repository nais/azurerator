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

func PolicyInPolicies(validPolicy ValidPolicy, policies []msgraph.ClaimsMappingPolicy) bool {
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
