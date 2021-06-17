package claimsmappingpolicy

import (
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
)

type ValidPolicies struct {
	NavIdent        ValidPolicy
	AzpName         ValidPolicy
	AllCustomClaims ValidPolicy
}

func (v ValidPolicies) All() []ValidPolicy {
	return []ValidPolicy{v.NavIdent, v.AzpName, v.AllCustomClaims}
}

type ValidPolicy struct {
	Name     v1.AzureAdExtraClaim
	ID       string
	Assigned bool
	Desired  bool
}
