package synchronizer

import (
	"context"
	"testing"
	"time"

	cache "github.com/Code-Hex/go-generics-cache"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	azurefake "github.com/nais/azureator/pkg/azure/fake"
	fakeazure "github.com/nais/azureator/pkg/azure/fake/client"
)

const (
	testTenantID    = "tenant-a"
	testClusterName = "test"
)

func newTestSweeper() *Sweeper {
	return &Sweeper{
		clusterName:   testClusterName,
		azureClient:   fakeazure.NewFakeAzureClient(),
		azureTenantID: testTenantID,
		cacheTTL:      time.Minute,
		resolveCache:  cache.New[string, resolved](),
		logger:        log.NewEntry(log.StandardLogger()),
	}
}

func TestSweeper_shouldResync(t *testing.T) {
	rule := &v1.AccessPolicyRule{Application: "consumer", Namespace: "team", Cluster: testClusterName}

	appWith := func(tenant string) v1.AzureAdApplication {
		return v1.AzureAdApplication{
			Status: v1.AzureAdApplicationStatus{
				SynchronizationTenant: tenant,
				PreAuthorizedApps: &v1.AzureAdPreAuthorizedAppsStatus{
					Unassigned: []v1.AzureAdPreAuthorizedApp{{AccessPolicyRule: rule}},
				},
			},
		}
	}

	tests := []struct {
		name string
		app  v1.AzureAdApplication
		want bool
	}{
		{
			name: "matching tenant is a candidate",
			app:  appWith(testTenantID),
			want: true,
		},
		{
			name: "different tenant is skipped",
			app:  appWith("tenant-b"),
			want: false,
		},
		{
			name: "empty tenant is skipped",
			app:  appWith(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSweeper()

			got := s.shouldResync(context.Background(), tt.app)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSweeper_shouldResync_staleAssigned(t *testing.T) {
	otherCluster := &v1.AccessPolicyRule{Application: "consumer", Namespace: "team", Cluster: "other"}
	sameCluster := &v1.AccessPolicyRule{Application: "consumer", Namespace: "team", Cluster: testClusterName}

	// the fake azure client resolves a rule to a deterministic client ID.
	liveClientID := azurefake.ClientIDForRule(*otherCluster)

	appWith := func(assigned v1.AzureAdPreAuthorizedApp) v1.AzureAdApplication {
		return v1.AzureAdApplication{
			Status: v1.AzureAdApplicationStatus{
				SynchronizationTenant: testTenantID,
				PreAuthorizedApps: &v1.AzureAdPreAuthorizedAppsStatus{
					Assigned: []v1.AzureAdPreAuthorizedApp{assigned},
				},
			},
		}
	}

	tests := []struct {
		name string
		app  v1.AzureAdApplication
		want bool
	}{
		{
			name: "cross-cluster assigned with stale client ID is a candidate",
			app:  appWith(v1.AzureAdPreAuthorizedApp{AccessPolicyRule: otherCluster, ClientID: "stale-client-id"}),
			want: true,
		},
		{
			name: "cross-cluster assigned with current client ID is skipped",
			app:  appWith(v1.AzureAdPreAuthorizedApp{AccessPolicyRule: otherCluster, ClientID: liveClientID}),
			want: false,
		},
		{
			name: "same-cluster assigned with stale client ID is skipped",
			app:  appWith(v1.AzureAdPreAuthorizedApp{AccessPolicyRule: sameCluster, ClientID: "stale-client-id"}),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSweeper()

			got := s.shouldResync(context.Background(), tt.app)
			assert.Equal(t, tt.want, got)
		})
	}
}
