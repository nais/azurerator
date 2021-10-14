package application_test

import (
	"github.com/nais/azureator/pkg/azure/client/application"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOptionalClaims_DescribeCreate(t *testing.T) {
	desired := msgraph.OptionalClaims{
		IDToken: []msgraph.OptionalClaim{
			{
				Essential: ptr.Bool(true),
				Name:      ptr.String("sid"),
			},
		},
	}

	create := application.Application{}.OptionalClaims().DescribeCreate()
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
				IDToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(false),
						Name:      ptr.String("upn"),
					},
				},
			},
			want: msgraph.OptionalClaims{
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
				IDToken: []msgraph.OptionalClaim{
					{
						Essential: ptr.Bool(false),
						Name:      ptr.String("sid"),
					},
				},
			},
			want: msgraph.OptionalClaims{
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
			actual := application.Application{}.OptionalClaims().DescribeUpdate(*existingApp)
			assert.Equal(t, test.want, *actual)
		})
	}
}
