package synchronizer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/event"
	"github.com/nais/azureator/pkg/kafka"
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

		e := &event.Event{}
		if err := json.Unmarshal(msg.Value, &e); err != nil {
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
			logger.Debugf("ignoring kafka event in same cluster '%s'", s.clusterName)
			return false, nil
		}

		if err := s.process(context.Background(), *e, logger); err != nil {
			return true, fmt.Errorf("processing event: %w", err)
		}

		return false, nil
	}
}

func (s Synchronizer) Local(ctx context.Context, e event.Event, logger *log.Entry) error {
	return s.process(ctx, e, logger)
}

func (s Synchronizer) process(ctx context.Context, e event.Event, logger *log.Entry) error {
	if !e.IsCreated() {
		logger.Debugf("ignoring event '%s'", e)
		return nil
	}

	logger.Infof("processing event '%s' for '%s'...", e, e.Application)

	var apps v1.AzureAdApplicationList
	err := s.reader.List(ctx, &apps)
	if err != nil {
		return fmt.Errorf("fetching AzureAdApplications from cluster: %w", err)
	}

	candidateCount := 0
	for _, app := range apps.Items {
		if hasMatchingPreAuthorizedApp(app, s.clusterName, e) {
			candidateID := kubernetes.UniformResourceName(&app, s.clusterName)
			candidateCount += 1

			if err := s.resync(ctx, app, e); err != nil {
				return fmt.Errorf("resyncing %s: %w", candidateID, err)
			}

			logger.Infof("marked '%s' for resync", candidateID)
		}
	}

	if candidateCount > 0 {
		logger.Infof("found and marked %d candidates for resync", candidateCount)
	} else {
		logger.Infof("no candidates found for resync")
	}
	return nil
}

func (s Synchronizer) resync(ctx context.Context, app v1.AzureAdApplication, e event.Event) error {
	existing := &v1.AzureAdApplication{}
	key := client.ObjectKey{Namespace: app.Namespace, Name: app.Name}

	if err := s.reader.Get(ctx, key, existing); err != nil {
		return fmt.Errorf("getting newest version from cluster: %s", err)
	}

	annotations.AddToAnnotation(existing, annotations.ResynchronizeKey, e.Application.String())
	annotations.AddToAnnotation(existing, v1.DeploymentCorrelationIDAnnotation, e.ID)

	if err := s.client.Update(ctx, existing); err != nil {
		return fmt.Errorf("setting resync annotation: %w", err)
	}

	return nil
}

func hasMatchingPreAuthorizedApp(in v1.AzureAdApplication, clusterName string, e event.Event) bool {
	for _, preAuthApp := range in.Spec.PreAuthorizedApplications {
		if len(preAuthApp.Namespace) == 0 {
			preAuthApp.Namespace = in.GetNamespace()
		}
		if len(preAuthApp.Cluster) == 0 {
			preAuthApp.Cluster = clusterName
		}

		nameMatches := preAuthApp.Application == e.Application.Name
		namespaceMatches := preAuthApp.Namespace == e.Application.Namespace
		clusterMatches := preAuthApp.Cluster == e.Application.Cluster

		if nameMatches && namespaceMatches && clusterMatches {
			return true
		}
	}

	return false
}
