package synchronizer

import (
	"testing"

	"github.com/nais/azureator/pkg/fixtures"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNeedsResync(t *testing.T) {
	clusterName := "test-cluster"
	clientID := "some-client-id"
	e := NewCreatedEvent("1", &metav1.ObjectMeta{
		Name:      "some-app",
		Namespace: "test-namespace",
	}, clusterName, clientID)

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

			actual := needsResync(*app, clusterName, e)
			if test.expected {
				assert.True(t, actual)
			} else {
				assert.False(t, actual)
			}
		})
	}
}

func TestNeedsResync_SelfReference(t *testing.T) {
	clusterName := "test-cluster"
	clientID := "some-client-id"

	// Event comes from the same app as the one being evaluated (test-app in test-namespace)
	e := NewCreatedEvent("1", &metav1.ObjectMeta{
		Name:      "test-app",
		Namespace: "test-namespace",
	}, clusterName, clientID)

	app := fixtures.MinimalApplication()
	app.Spec.PreAuthorizedApplications = []nais_io_v1.AccessPolicyInboundRule{{
		AccessPolicyRule: nais_io_v1.AccessPolicyRule{
			Application: "test-app",
			Namespace:   "test-namespace",
			Cluster:     clusterName,
		},
	}}

	assert.False(t, needsResync(*app, clusterName, e), "app should not resync itself")
}

func TestNeedsResync_AssignedStatus(t *testing.T) {
	clusterName := "test-cluster"
	matchingRule := nais_io_v1.AccessPolicyRule{
		Application: "some-app",
		Namespace:   "test-namespace",
		Cluster:     clusterName,
	}

	newEvent := func(factory func(string, metav1.Object, string, string) Event, clientID string) Event {
		return factory("1", &metav1.ObjectMeta{
			Name:      "some-app",
			Namespace: "test-namespace",
		}, clusterName, clientID)
	}

	withAssigned := func(clientID string, rule *nais_io_v1.AccessPolicyRule) *nais_io_v1.AzureAdApplication {
		app := fixtures.MinimalApplication()
		app.Spec.PreAuthorizedApplications = []nais_io_v1.AccessPolicyInboundRule{{AccessPolicyRule: matchingRule}}
		app.Status.PreAuthorizedApps = &nais_io_v1.AzureAdPreAuthorizedAppsStatus{
			Assigned: []nais_io_v1.AzureAdPreAuthorizedApp{{
				AccessPolicyRule: rule,
				ClientID:         clientID,
			}},
		}
		return app
	}

	for _, test := range []struct {
		name     string
		app      *nais_io_v1.AzureAdApplication
		event    Event
		expected bool
	}{
		{
			name:     "created event, not yet assigned",
			app:      fixtures.MinimalApplication(),
			event:    newEvent(NewCreatedEvent, "some-client-id"),
			expected: false, // no matching rule in spec -> no resync
		},
		{
			name: "created event, matching rule in spec, not assigned",
			app: func() *nais_io_v1.AzureAdApplication {
				app := fixtures.MinimalApplication()
				app.Spec.PreAuthorizedApplications = []nais_io_v1.AccessPolicyInboundRule{{AccessPolicyRule: matchingRule}}
				return app
			}(),
			event:    newEvent(NewCreatedEvent, "some-client-id"),
			expected: true,
		},
		{
			name:     "created event, assigned with matching ClientID",
			app:      withAssigned("some-client-id", &matchingRule),
			event:    newEvent(NewCreatedEvent, "some-client-id"),
			expected: false,
		},
		{
			name:     "created event, assigned with different ClientID",
			app:      withAssigned("old-client-id", &matchingRule),
			event:    newEvent(NewCreatedEvent, "new-client-id"),
			expected: true,
		},
		{
			name:     "updated event, assigned with matching ClientID",
			app:      withAssigned("some-client-id", &matchingRule),
			event:    newEvent(NewUpdatedEvent, "some-client-id"),
			expected: false,
		},
		{
			name:     "updated event, assigned with different ClientID",
			app:      withAssigned("old-client-id", &matchingRule),
			event:    newEvent(NewUpdatedEvent, "new-client-id"),
			expected: true,
		},
		{
			name:     "updated event, not yet assigned",
			app:      withAssigned("some-client-id", nil),
			event:    newEvent(NewUpdatedEvent, "some-client-id"),
			expected: true, // nil rule in status is skipped, matching spec rule triggers resync
		},
		{
			name: "nil AccessPolicyRule in status does not panic",
			app: func() *nais_io_v1.AzureAdApplication {
				app := withAssigned("some-client-id", nil)
				return app
			}(),
			event:    newEvent(NewCreatedEvent, "some-client-id"),
			expected: true,
		},
		{
			// Symmetry: spec rules get cluster/namespace defaulted; status rules must too,
			// otherwise a same-cluster/same-namespace producer with short-form status would
			// be treated as not-yet-assigned and trigger needless resyncs.
			name: "spec rule fully qualified, status rule short-form, same ClientID",
			app: func() *nais_io_v1.AzureAdApplication {
				app := fixtures.MinimalApplication()
				app.Spec.PreAuthorizedApplications = []nais_io_v1.AccessPolicyInboundRule{{AccessPolicyRule: matchingRule}}
				app.Status.PreAuthorizedApps = &nais_io_v1.AzureAdPreAuthorizedAppsStatus{
					Assigned: []nais_io_v1.AzureAdPreAuthorizedApp{{
						AccessPolicyRule: &nais_io_v1.AccessPolicyRule{Application: "some-app"}, // omitted ns + cluster
						ClientID:         "some-client-id",
					}},
				}
				return app
			}(),
			event:    newEvent(NewCreatedEvent, "some-client-id"),
			expected: false,
		},
		{
			// Empty ClientID on the event must never match an empty ClientID on a status
			// assignment — that would skip resync on a false equality.
			name:     "empty ClientID on event, empty ClientID assigned -> resync",
			app:      withAssigned("", &matchingRule),
			event:    newEvent(NewUpdatedEvent, ""),
			expected: true,
		},
		{
			name:     "empty ClientID on event, valid ClientID assigned -> resync",
			app:      withAssigned("some-client-id", &matchingRule),
			event:    newEvent(NewUpdatedEvent, ""),
			expected: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual := needsResync(*test.app, clusterName, test.event)
			assert.Equal(t, test.expected, actual)
		})
	}
}
