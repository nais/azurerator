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

const sourceSweep = "sweep"

var (
	_ manager.Runnable               = (*Sweeper)(nil)
	_ manager.LeaderElectionRunnable = (*Sweeper)(nil)
)

// Sweeper periodically marks AzureAdApplications with unassigned preAuthorizedApps for
// resync, catching cases missed by the event-driven path in [Synchronizer].
type Sweeper struct {
	clusterName     string
	kubeClient      client.Client
	reader          client.Reader
	azureClient     azure.Client
	interval        time.Duration
	cacheTTL        time.Duration
	assignableCache *cache.Cache[string, bool]
	logger          *log.Entry
}

func NewSweeper(clusterName string, kubeClient client.Client, reader client.Reader, azureClient azure.Client, interval time.Duration) *Sweeper {
	const minSweepInterval = time.Second
	interval = max(interval, minSweepInterval)
	cacheTTL := interval / 2

	return &Sweeper{
		clusterName:     clusterName,
		kubeClient:      kubeClient,
		reader:          reader,
		azureClient:     azureClient,
		interval:        interval,
		cacheTTL:        cacheTTL,
		assignableCache: cache.New[string, bool](),
		logger:          log.WithField("subsystem", "synchronizer/sweeper"),
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
		marked, err := s.resync(ctx, app)
		if err != nil {
			metrics.ResyncFailedTotal.WithLabelValues(app.Namespace, sourceSweep).Inc()
			s.logger.Errorf("marking %s for resync: %v", candidateID, err)
			continue
		}
		if !marked {
			continue
		}

		metrics.ResyncCandidatesTotal.WithLabelValues(app.Namespace, sourceSweep).Inc()
		candidateCount++

		s.logger.Debugf("marked '%s' for resync (has %d unassigned preAuthorizedApps)", candidateID, len(app.Status.PreAuthorizedApps.Unassigned))
	}

	metrics.ResyncFanout.WithLabelValues(sourceSweep).Observe(float64(candidateCount))

	if candidateCount > 0 {
		s.logger.Infof("sweep found and marked %d candidates for resync", candidateCount)
	} else {
		s.logger.Debugf("sweep completed, no candidates found")
	}
}

func (s *Sweeper) shouldResync(ctx context.Context, app v1.AzureAdApplication) bool {
	if app.Status.PreAuthorizedApps == nil || len(app.Status.PreAuthorizedApps.Unassigned) == 0 {
		return false
	}

	if _, hasPendingResyncAnnotation := annotations.HasAnnotation(&app, annotations.ResynchronizeKey); hasPendingResyncAnnotation {
		return false
	}

	return s.hasAssignableUnassignedPreAuthApp(ctx, app)
}

func (s *Sweeper) hasAssignableUnassignedPreAuthApp(ctx context.Context, app v1.AzureAdApplication) bool {
	for _, unassigned := range app.Status.PreAuthorizedApps.Unassigned {
		if unassigned.AccessPolicyRule == nil {
			continue
		}

		name := customresources.GetUniqueName(*unassigned.AccessPolicyRule)
		exists, cached := s.assignableCache.Get(name)
		if cached {
			if exists {
				return true
			}
			continue
		}

		exists, err := s.azureClient.PreAuthorizedAppCanBeAssigned(ctx, *unassigned.AccessPolicyRule)
		if err != nil {
			s.logger.Debugf("pre-flight check failed for %s: %v", name, err)
			continue
		}

		s.assignableCache.Set(name, exists, cache.WithExpiration(s.cacheTTL))
		if exists {
			return true
		}
	}

	return false
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
		annotations.SetAnnotation(existing, annotations.ResynchronizeKey, sourceSweep)

		if err := s.kubeClient.Update(ctx, existing); err != nil {
			return fmt.Errorf("setting resync annotation: %w", err)
		}
		marked = true
		return nil
	})

	return marked, err
}
