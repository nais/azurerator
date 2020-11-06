package claimsmappingpolicy

import "fmt"

type Payload struct {
	Content string `json:"@odata.id"`
}

func ToClaimsMappingPolicyPayload(policyId string) Payload {
	return Payload{
		Content: fmt.Sprintf("https://graph.microsoft.com/v1.0/policies/claimsMappingPolicies/%s", policyId),
	}
}
