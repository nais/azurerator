package synchronizer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	nais_io "github.com/nais/liberator/pkg/apis/nais.io"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/kafka"
	"github.com/nais/azureator/pkg/metrics"
)

const (
	sourceLocal = "local"
	sourceKafka = "kafka"

	resultProcessed    = "processed"
	resultIgnored      = "ignored"
	resultInvalid      = "invalid"
	resultCrossCluster = "cross_cluster"
	resultUnmarshal    = "unmarshal_error"
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
		clusterName,
		client,
		reader,
	}
}

// Kafka processes incoming Kafka messages and triggers resynchronization of Azure AD applications as needed.
func (s Synchronizer) Kafka() kafka.Callback {
	return func(msg *sarama.ConsumerMessage, logger *log.Entry) (bool, error) {
		logger.Debugf("incoming message from Kafka")

		e := &Event{}
		if err := json.Unmarshal(msg.Value, &e); err != nil {
			metrics.ResyncEventsTotal.WithLabelValues(sourceKafka, "", resultUnmarshal).Inc()
			return false, fmt.Errorf("unmarshalling message to event; ignoring: %w", err)
		}

		logger = logger.WithFields(log.Fields{
			"CorrelationID":         e.ID,
			"application_name":      e.Application.Name,
			"application_namespace": e.Application.Namespace,
			"application_cluster":   e.Application.Cluster,
			"event_name":            e.Name,
		})

		if e.Application.Cluster == s.clusterName {
			// events targeting this cluster is handled by [Synchronizer.Local]
			metrics.ResyncEventsTotal.WithLabelValues(sourceKafka, string(e.Name), resultCrossCluster).Inc()
			logger.Debugf("ignoring kafka event in same cluster '%s'", s.clusterName)
			return false, nil
		}

		if err := s.process(context.Background(), sourceKafka, *e, logger); err != nil {
			return true, fmt.Errorf("processing event: %w", err)
		}

		return false, nil
	}
}

func (s Synchronizer) Local(ctx context.Context, e Event, logger *log.Entry) error {
	return s.process(ctx, sourceLocal, e, logger)
}

func (s Synchronizer) process(ctx context.Context, source string, e Event, logger *log.Entry) error {
	// Delete events are not propagated: producers do not emit them, and consumers
	// converge their preauth status against spec on their own reconcile loop.
	if !e.IsCreated() && !e.IsUpdated() {
		metrics.ResyncEventsTotal.WithLabelValues(source, string(e.Name), resultIgnored).Inc()
		logger.Debugf("ignoring event '%s'", e)
		return nil
	}

	if err := e.Validate(); err != nil {
		metrics.ResyncEventsTotal.WithLabelValues(source, string(e.Name), resultInvalid).Inc()
		logger.Warnf("ignoring event '%s' for '%s': %v", e, e.Application, err)
		return nil
	}

	metrics.ResyncEventsTotal.WithLabelValues(source, string(e.Name), resultProcessed).Inc()
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
			candidateCount += 1

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

	// Retry on conflict: concurrent events (e.g. Kafka and Local) may race on the same
	// consumer, and we'd rather retry than silently drop a resync request.
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing := &v1.AzureAdApplication{}
		if err := s.reader.Get(ctx, key, existing); err != nil {
			return fmt.Errorf("getting newest version from cluster: %s", err)
		}

		annotations.AddToAnnotation(existing, annotations.ResynchronizeKey, e.Application.String())
		// Correlation ID is a single value, not a queue: overwrite to avoid unbounded growth
		// (annotations are capped at 256 KiB by Kubernetes).
		annotations.SetAnnotation(existing, nais_io.DeploymentCorrelationIDAnnotation, e.ID)

		if err := s.client.Update(ctx, existing); err != nil {
			return fmt.Errorf("setting resync annotation: %w", err)
		}
		return nil
	})
}

func needsResync(in v1.AzureAdApplication, clusterName string, e Event) bool {
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
