package claimsmappingpolicy

import v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

type ValidPolicies struct {
	Policies []ValidPolicy
}

func (v ValidPolicies) HasPolicyByID(id string) (ValidPolicy, bool) {
	return v.hasPolicy(func(policy ValidPolicy) bool {
		return policy.ID == id
	})
}

func (v ValidPolicies) HasPolicyByName(name v1.AzureAdExtraClaim) (ValidPolicy, bool) {
	return v.hasPolicy(func(policy ValidPolicy) bool {
		return policy.Name == name
	})
}

func (v ValidPolicies) hasPolicy(cond func(policy ValidPolicy) bool) (ValidPolicy, bool) {
	for _, p := range v.Policies {
		if cond(p) {
			return p, true
		}
	}
	return ValidPolicy{}, false
}

type ValidPolicy struct {
	Name v1.AzureAdExtraClaim
	ID   string
}
