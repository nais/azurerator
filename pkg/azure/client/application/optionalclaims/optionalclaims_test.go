package optionalclaims_test

import (
	"testing"

	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure/client/application/optionalclaims"
	"github.com/nais/azureator/pkg/azure/util"
)

func TestOptionalClaims_DescribeCreate(t *testing.T) {
	desired := msgraph.OptionalClaims{
		AccessToken: []msgraph.OptionalClaim{
			{
				Essential: ptr.Bool(true),
				Name:      ptr.String("idtyp"),
			},
		},
		IDToken: []msgraph.OptionalClaim{
			{
				Essential: ptr.Bool(true),
				Name:      ptr.String("sid"),
			},
		},
	}

	create := optionalclaims.NewOptionalClaims().DescribeCreate()
	assert.Equal(t, desired, *create)
}

func TestOptionalClaims_DescribeUpdate(t *testing.T) {
	for _, test := range []struct {
		name     string
		existing msgraph.OptionalClaims
		want     msgraph.OptionalClaims
	}{
		{
			name:     "no existing optional claims",
			existing: msgraph.OptionalClaims{},
			want: msgraph.OptionalClaims{
				AccessToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(true),
						Name:      ptr.String("idtyp"),
					},
				},
				IDToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(true),
						Name:      ptr.String("sid"),
					},
				},
			},
		},
		{
			name: "existing non-conflicting optional claims",
			existing: msgraph.OptionalClaims{
				AccessToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(true),
						Name:      ptr.String("upn"),
					},
				},
				IDToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(false),
						Name:      ptr.String("upn"),
					},
				},
			},
			want: msgraph.OptionalClaims{
				AccessToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(true),
						Name:      ptr.String("upn"),
					},
					{
						Essential: ptr.Bool(true),
						Name:      ptr.String("idtyp"),
					},
				},
				IDToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(false),
						Name:      ptr.String("upn"),
					},
					{
						Essential: ptr.Bool(true),
						Name:      ptr.String("sid"),
					},
				},
			},
		},
		{
			name: "existing conflicting optional claims",
			existing: msgraph.OptionalClaims{
				AccessToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(true),
						Name:      ptr.String("idtyp"),
					},
				},
				IDToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(false),
						Name:      ptr.String("sid"),
					},
				},
			},
			want: msgraph.OptionalClaims{
				AccessToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(true),
						Name:      ptr.String("idtyp"),
					},
				},
				IDToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(true),
						Name:      ptr.String("sid"),
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			existingApp := util.EmptyApplication().OptionalClaims(&test.existing).Build()
			actual := optionalclaims.NewOptionalClaims().DescribeUpdate(*existingApp)
			assert.Equal(t, test.want, *actual)
		})
	}
}
