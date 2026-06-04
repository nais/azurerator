package synchronizer

import (
	"context"
	"testing"
	"time"

	cache "github.com/Code-Hex/go-generics-cache"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	fakeazure "github.com/nais/azureator/pkg/azure/fake/client"
)

const testTenantID = "tenant-a"

func TestSweeper_shouldResync(t *testing.T) {
	rule := &v1.AccessPolicyRule{Application: "consumer", Namespace: "team", Cluster: "test"}

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
			s := &Sweeper{
				azureClient:     fakeazure.NewFakeAzureClient(),
				azureTenantID:   testTenantID,
				cacheTTL:        time.Minute,
				assignableCache: cache.New[string, bool](),
				logger:          log.NewEntry(log.StandardLogger()),
			}

			got := s.shouldResync(context.Background(), tt.app)
			assert.Equal(t, tt.want, got)
		})
	}
}
