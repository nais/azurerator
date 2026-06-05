package synchronizer

import (
	"context"
	"fmt"

	nais_io "github.com/nais/liberator/pkg/apis/nais.io"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/metrics"
)

const (
	sourceSynchronizer = "synchronizer"

	resultProcessed = "processed"
	resultIgnored   = "ignored"
	resultInvalid   = "invalid"
)

// Synchronizer ensures that the Azure AD applications are resynchronized on relevant events,
// e.g. on creation of previously non-existing pre-authorized applications.
type Synchronizer struct {
	clusterName string
	client      client.Client
	reader      client.Reader
}

func New(clusterName string, client client.Client, reader client.Reader) *Synchronizer {
	return &Synchronizer{
		clusterName: clusterName,
		client:      client,
		reader:      reader,
	}
}

func (s Synchronizer) Synchronize(ctx context.Context, e Event, logger *log.Entry) error {
	logger = logger.WithField("subsystem", sourceSynchronizer)

	// Delete events are not propagated: producers do not emit them, and consumers
	// converge their preauth status against spec on their own reconcile loop.
	if !e.IsCreated() && !e.IsUpdated() {
		metrics.ResyncEventsTotal.WithLabelValues(sourceSynchronizer, string(e.Name), resultIgnored).Inc()
		logger.Debugf("ignoring event '%s'", e)
		return nil
	}

	if err := e.Validate(); err != nil {
		metrics.ResyncEventsTotal.WithLabelValues(sourceSynchronizer, string(e.Name), resultInvalid).Inc()
		logger.Warnf("ignoring event '%s' for '%s': %v", e, e.Application, err)
		return nil
	}

	metrics.ResyncEventsTotal.WithLabelValues(sourceSynchronizer, string(e.Name), resultProcessed).Inc()
	logger.Infof("processing event '%s' for '%s'...", e, e.Application)

	var apps v1.AzureAdApplicationList
	err := s.reader.List(ctx, &apps)
	if err != nil {
		return fmt.Errorf("fetching AzureAdApplications from cluster: %w", err)
	}

	candidateCount := 0
	for _, app := range apps.Items {
		if needsResync(app, s.clusterName, e) {
			candidateID := kubernetes.UniformResourceName(&app, s.clusterName)

			if err := s.resync(ctx, app, e); err != nil {
				metrics.ResyncFailedTotal.WithLabelValues(app.Namespace, string(e.Name)).Inc()
				return fmt.Errorf("resyncing %s: %w", candidateID, err)
			}

			metrics.ResyncCandidatesTotal.WithLabelValues(app.Namespace, string(e.Name)).Inc()
			candidateCount++

			logger.Infof("marked '%s' for resync", candidateID)
		}
	}

	metrics.ResyncFanout.WithLabelValues(string(e.Name)).Observe(float64(candidateCount))

	if candidateCount > 0 {
		logger.Infof("found and marked %d candidates for resync", candidateCount)
	} else {
		logger.Infof("no candidates found for resync")
	}
	return nil
}

func (s Synchronizer) resync(ctx context.Context, app v1.AzureAdApplication, e Event) error {
	key := client.ObjectKey{Namespace: app.Namespace, Name: app.Name}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing := &v1.AzureAdApplication{}
		if err := s.reader.Get(ctx, key, existing); err != nil {
			return fmt.Errorf("getting newest version from cluster: %w", err)
		}

		annotations.AddToAnnotation(existing, annotations.ResynchronizeKey, e.Application.String())
		annotations.SetAnnotation(existing, nais_io.DeploymentCorrelationIDAnnotation, e.ID)

		if err := s.client.Update(ctx, existing); err != nil {
			return fmt.Errorf("setting resync annotation: %w", err)
		}
		return nil
	})
}

func needsResync(in v1.AzureAdApplication, clusterName string, e Event) bool {
	// An application must never resync itself (e.g. when it lists itself in its own preAuthorizedApps).
	if in.GetName() == e.Application.Name &&
		in.GetNamespace() == e.Application.Namespace &&
		clusterName == e.Application.Cluster {
		return false
	}

	normalize := func(rule v1.AccessPolicyRule) v1.AccessPolicyRule {
		if len(rule.Namespace) == 0 {
			rule.Namespace = in.GetNamespace()
		}
		if len(rule.Cluster) == 0 {
			rule.Cluster = clusterName
		}
		return rule
	}

	matches := func(rule v1.AccessPolicyRule) bool {
		rule = normalize(rule)
		return rule.Application == e.Application.Name &&
			rule.Namespace == e.Application.Namespace &&
			rule.Cluster == e.Application.Cluster
	}

	alreadyAssignedWithCurrentClientID := func() bool {
		if e.Application.ClientID == "" || in.Status.PreAuthorizedApps == nil {
			return false
		}
		for _, assigned := range in.Status.PreAuthorizedApps.Assigned {
			if assigned.AccessPolicyRule == nil {
				continue
			}
			if matches(*assigned.AccessPolicyRule) && assigned.ClientID == e.Application.ClientID {
				return true
			}
		}
		return false
	}

	for _, preAuthApp := range in.Spec.PreAuthorizedApplications {
		if matches(preAuthApp.AccessPolicyRule) {
			return !alreadyAssignedWithCurrentClientID()
		}
	}
	return false
}
