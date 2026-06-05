package synchronizer

import (
	"context"
	"fmt"
	"time"

	cache "github.com/Code-Hex/go-generics-cache"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/customresources"
	"github.com/nais/azureator/pkg/metrics"
)

const sourceSweeper = "sweeper"

var (
	_ manager.Runnable               = (*Sweeper)(nil)
	_ manager.LeaderElectionRunnable = (*Sweeper)(nil)
)

// resolved is the cached outcome of resolving a pre-authorized app in Azure AD.
type resolved struct {
	clientID   string
	assignable bool
}

// Sweeper periodically marks AzureAdApplications for resync, catching cases missed by the
// event-driven path in [Synchronizer]. It handles two cases:
//   - pre-authorized apps that were unassignable during reconcile but have since appeared, and
//   - cross-cluster pre-authorized apps that were recreated with a new client ID (same-cluster
//     client ID changes are handled immediately by [Synchronizer]).
type Sweeper struct {
	clusterName   string
	kubeClient    client.Client
	reader        client.Reader
	azureClient   azure.Client
	azureTenantID string
	interval      time.Duration
	cacheTTL      time.Duration
	resolveCache  *cache.Cache[string, resolved]
	logger        *log.Entry
}

func NewSweeper(
	clusterName string,
	kubeClient client.Client,
	reader client.Reader,
	azureClient azure.Client,
	azureTenantID string,
	interval time.Duration,
) *Sweeper {
	const minSweepInterval = time.Second
	interval = max(interval, minSweepInterval)
	cacheTTL := interval / 2

	return &Sweeper{
		clusterName:   clusterName,
		kubeClient:    kubeClient,
		reader:        reader,
		azureClient:   azureClient,
		azureTenantID: azureTenantID,
		interval:      interval,
		cacheTTL:      cacheTTL,
		resolveCache:  cache.New[string, resolved](),
		logger:        log.WithField("subsystem", sourceSweeper),
	}
}

func (s *Sweeper) Start(ctx context.Context) error {
	s.logger.Infof("starting periodic sweep every %s", s.interval)

	t := time.NewTicker(s.interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("stopping periodic sweep")
			return nil
		case <-t.C:
			s.sweep(ctx)
		}
	}
}

func (s *Sweeper) NeedLeaderElection() bool {
	return true
}

func (s *Sweeper) sweep(ctx context.Context) {
	var apps v1.AzureAdApplicationList
	if err := s.reader.List(ctx, &apps); err != nil {
		s.logger.Errorf("listing AzureAdApplications: %v", err)
		return
	}

	candidateCount := 0
	for _, app := range apps.Items {
		if !s.shouldResync(ctx, app) {
			continue
		}

		candidateID := kubernetes.UniformResourceName(&app, s.clusterName)
		metrics.ResyncEventsTotal.WithLabelValues(sourceSweeper, candidateID, resultProcessed).Inc()

		marked, err := s.resync(ctx, app)
		if err != nil {
			metrics.ResyncFailedTotal.WithLabelValues(app.Namespace, sourceSweeper).Inc()
			s.logger.Errorf("marking %s for resync: %v", candidateID, err)
			continue
		}
		if !marked {
			continue
		}

		metrics.ResyncCandidatesTotal.WithLabelValues(app.Namespace, candidateID).Inc()
		candidateCount++

		s.logger.Debugf("marked '%s' for resync", candidateID)
	}

	metrics.ResyncFanout.WithLabelValues(sourceSweeper).Observe(float64(candidateCount))

	if candidateCount > 0 {
		s.logger.Infof("sweep found and marked %d candidates for resync", candidateCount)
	} else {
		s.logger.Debugf("sweep completed, no candidates found")
	}
}

func (s *Sweeper) shouldResync(ctx context.Context, app v1.AzureAdApplication) bool {
	if app.Status.PreAuthorizedApps == nil {
		return false
	}

	if app.Status.SynchronizationTenant != s.azureTenantID {
		return false
	}

	if _, hasPendingResyncAnnotation := annotations.HasAnnotation(&app, annotations.ResynchronizeKey); hasPendingResyncAnnotation {
		return false
	}

	return s.hasResyncablePreAuthApp(ctx, app)
}

// hasResyncablePreAuthApp reports whether the app has a pre-authorized app that warrants a resync:
// an unassigned entry that has since become assignable, or a cross-cluster assigned entry whose
// app was recreated with a new client ID.
func (s *Sweeper) hasResyncablePreAuthApp(ctx context.Context, app v1.AzureAdApplication) bool {
	for _, unassigned := range app.Status.PreAuthorizedApps.Unassigned {
		if unassigned.AccessPolicyRule == nil {
			continue
		}

		r, ok := s.resolve(ctx, *unassigned.AccessPolicyRule)
		if ok && r.assignable {
			return true
		}
	}

	for _, assigned := range app.Status.PreAuthorizedApps.Assigned {
		if assigned.AccessPolicyRule == nil {
			continue
		}

		// same-cluster client ID changes are handled immediately by [Synchronizer].
		if assigned.AccessPolicyRule.Cluster == s.clusterName {
			continue
		}

		r, ok := s.resolve(ctx, *assigned.AccessPolicyRule)
		if ok && r.assignable && r.clientID != assigned.ClientID {
			return true
		}
	}

	return false
}

// resolve looks up the live state of a pre-authorized app, caching the outcome for [Sweeper.cacheTTL].
// The second return value is false when the lookup was inconclusive (e.g. a transient Azure error).
func (s *Sweeper) resolve(ctx context.Context, rule v1.AccessPolicyRule) (resolved, bool) {
	name := customresources.GetUniqueName(rule)
	if r, cached := s.resolveCache.Get(name); cached {
		return r, true
	}

	clientID, assignable, err := s.azureClient.PreAuthorizedAppClientID(ctx, rule)
	if err != nil {
		s.logger.Debugf("pre-flight check failed for %s: %v", name, err)
		return resolved{}, false
	}

	r := resolved{clientID: clientID, assignable: assignable}
	s.resolveCache.Set(name, r, cache.WithExpiration(s.cacheTTL))
	return r, true
}

func (s *Sweeper) resync(ctx context.Context, app v1.AzureAdApplication) (bool, error) {
	key := client.ObjectKey{Namespace: app.Namespace, Name: app.Name}
	marked := false

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing := &v1.AzureAdApplication{}
		if err := s.reader.Get(ctx, key, existing); err != nil {
			return fmt.Errorf("getting newest version from cluster: %w", err)
		}

		if _, hasPending := annotations.HasAnnotation(existing, annotations.ResynchronizeKey); hasPending {
			return nil
		}
		annotations.SetAnnotation(existing, annotations.ResynchronizeKey, sourceSweeper)

		if err := s.kubeClient.Update(ctx, existing); err != nil {
			return fmt.Errorf("setting resync annotation: %w", err)
		}
		marked = true
		return nil
	})

	return marked, err
}
