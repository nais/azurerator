package optionalclaims

import (
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
)

type OptionalClaims interface {
	DescribeCreate() *msgraph.OptionalClaims
	DescribeUpdate(existing msgraph.Application) *msgraph.OptionalClaims
}

type optionalClaims struct{}

func NewOptionalClaims() OptionalClaims {
	return optionalClaims{}
}

func (o optionalClaims) DescribeCreate() *msgraph.OptionalClaims {
	return defaultClaims()
}

func (o optionalClaims) DescribeUpdate(existing msgraph.Application) *msgraph.OptionalClaims {
	existingClaims := existing.OptionalClaims
	if existingClaims == nil {
		return defaultClaims()
	}

	return mergeClaims(existingClaims, defaultClaims())
}

func defaultClaims() *msgraph.OptionalClaims {
	return &msgraph.OptionalClaims{
		IDToken: defaultIDTokenClaims(),
	}
}

func defaultIDTokenClaims() []msgraph.OptionalClaim {
	return []msgraph.OptionalClaim{
		{
			Essential: ptr.Bool(true),
			Name:      ptr.String("sid"),
		},
	}
}

func mergeClaims(existing, override *msgraph.OptionalClaims) *msgraph.OptionalClaims {
	result := *existing

	merge := func(existing, override []msgraph.OptionalClaim) []msgraph.OptionalClaim {
		for _, overrideClaim := range override {
			seen := false

			for i, claim := range existing {
				if claim.Name == nil || overrideClaim.Name == nil {
					continue
				}

				if *claim.Name == *overrideClaim.Name {
					existing[i] = overrideClaim
					seen = true
				}
			}

			if !seen {
				existing = append(existing, overrideClaim)
			}
		}

		return existing
	}

	if len(override.IDToken) > 0 {
		result.IDToken = merge(result.IDToken, override.IDToken)
	}

	if len(override.AccessToken) > 0 {
		result.AccessToken = merge(result.AccessToken, override.AccessToken)
	}

	if len(override.Saml2Token) > 0 {
		result.Saml2Token = merge(result.Saml2Token, override.Saml2Token)
	}

	return &result
}
