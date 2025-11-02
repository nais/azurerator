package synchronizer

import (
	"testing"

	"github.com/nais/azureator/pkg/event"
	"github.com/nais/azureator/pkg/fixtures"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHasMatchingPreAuthorizedApp(t *testing.T) {
	clusterName := "test-cluster"
	e := event.New("1", event.Created, &metav1.ObjectMeta{
		Name:      "some-app",
		Namespace: "test-namespace",
	}, clusterName)

	for _, test := range []struct {
		name     string
		rule     nais_io_v1.AccessPolicyRule
		expected bool
	}{
		{
			name:     "no rule",
			rule:     nais_io_v1.AccessPolicyRule{},
			expected: false,
		},
		{
			name: "non-matching app",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "another-app",
			},
			expected: false,
		},
		{
			name: "non-matching namespace",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Namespace:   "another-namespace",
			},
			expected: false,
		},
		{
			name: "non-matching cluster",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Cluster:     "another-cluster",
			},
			expected: false,
		},
		{
			name: "non-matching namespace and cluster",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Namespace:   "another-namespace",
				Cluster:     "another-cluster",
			},
			expected: false,
		},
		{
			name: "no matching fields",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "another-app",
				Namespace:   "another-namespace",
				Cluster:     "another-cluster",
			},
			expected: false,
		},
		{
			name: "all fields matching",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Namespace:   "test-namespace",
				Cluster:     "test-cluster",
			},
			expected: true,
		},
		{
			name: "matching app and namespace, omitted cluster",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Namespace:   "test-namespace",
			},
			expected: true,
		},
		{
			name: "matching app and cluster, omitted namespace",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
				Cluster:     "test-cluster",
			},
			expected: true,
		},
		{
			name: "matching app, omitted cluster and namespace",
			rule: nais_io_v1.AccessPolicyRule{
				Application: "some-app",
			},
			expected: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := fixtures.MinimalApplication()
			app.Spec.PreAuthorizedApplications = []nais_io_v1.AccessPolicyInboundRule{{AccessPolicyRule: test.rule}}

			actual := hasMatchingPreAuthorizedApp(*app, clusterName, e)
			if test.expected {
				assert.True(t, actual)
			} else {
				assert.False(t, actual)
			}
		})
	}
}
