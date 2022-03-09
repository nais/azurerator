package identifieruri_test

import (
	"testing"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application/identifieruri"
)

func TestDescribeCreate(t *testing.T) {
	spec := spec()
	actual := identifieruri.DescribeCreate(spec)
	expected := azure.IdentifierUris{
		"api://test-cluster.test-namespace.test",
		"api://some-uuid",
	}

	assert.ElementsMatch(t, expected, actual)
}

func TestDescribeUpdate(t *testing.T) {
	for _, test := range []struct {
		name     string
		existing azure.IdentifierUris
		expected azure.IdentifierUris
	}{
		{
			name:     "no existing uris",
			existing: nil,
			expected: azure.IdentifierUris{
				"api://test-cluster.test-namespace.test",
				"api://some-uuid",
			},
		},
		{
			name: "existing uris, no overlap with default",
			existing: azure.IdentifierUris{
				"api://some-other-uri",
			},
			expected: azure.IdentifierUris{
				"api://some-other-uri",
				"api://test-cluster.test-namespace.test",
				"api://some-uuid",
			},
		},
		{
			name: "existing uris, partial overlap with default",
			existing: azure.IdentifierUris{
				"api://some-other-uri",
				"api://test-cluster.test-namespace.test",
			},
			expected: azure.IdentifierUris{
				"api://some-other-uri",
				"api://test-cluster.test-namespace.test",
				"api://some-uuid",
			},
		},
		{
			name: "existing uris, full overlap with default",
			existing: azure.IdentifierUris{
				"api://some-other-uri",
				"api://test-cluster.test-namespace.test",
				"api://some-uuid",
			},
			expected: azure.IdentifierUris{
				"api://some-other-uri",
				"api://test-cluster.test-namespace.test",
				"api://some-uuid",
			},
		},
		{
			name: "existing uris, equal to default",
			existing: azure.IdentifierUris{
				"api://test-cluster.test-namespace.test",
				"api://some-uuid",
			},
			expected: azure.IdentifierUris{
				"api://test-cluster.test-namespace.test",
				"api://some-uuid",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec := spec()
			actual := identifieruri.DescribeUpdate(spec, test.existing)
			assert.ElementsMatch(t, test.expected, actual)
		})
	}
}

func spec() v1.AzureAdApplication {
	spec := v1.AzureAdApplication{}
	spec.SetName("test")
	spec.SetNamespace("test-namespace")
	spec.SetClusterName("test-cluster")
	spec.Status.ClientId = "some-uuid"
	return spec
}
