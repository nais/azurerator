package synchronizer

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Shopify/sarama"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/customresources"
	"github.com/nais/azureator/pkg/event"
	"github.com/nais/azureator/pkg/kafka"
)

// Synchronizer ensures that the Azure AD applications between multiple Azurerator instances/clusters are up-to-date and eventually consistent.
type Synchronizer interface {
	Callback() kafka.Callback
}

type synchronizer struct {
	config     config.Config
	kubeClient client.Client
	kubeReader client.Reader
}

func NewSynchronizer(config config.Config, kubeClient client.Client, kubeReader client.Reader) Synchronizer {
	return synchronizer{
		kubeClient: kubeClient,
		kubeReader: kubeReader,
		config:     config,
	}
}

func (s synchronizer) Callback() kafka.Callback {
	return func(msg *sarama.ConsumerMessage, logger *log.Entry) (bool, error) {
		logger.Debugf("incoming message from Kafka")

		eventMsg := &event.Event{}
		if err := json.Unmarshal(msg.Value, &eventMsg); err != nil {
			return false, fmt.Errorf("unmarshalling message to event; ignoring: %w", err)
		}

		logger = logger.WithFields(log.Fields{
			"CorrelationID": eventMsg.ID,
			"application":   eventMsg.Application,
			"event_name":    eventMsg.EventName,
		})

		if err := s.process(context.Background(), *eventMsg, logger); err != nil {
			return true, fmt.Errorf("processing event: %w", err)
		}

		logger.Info("event synchronized")
		return false, nil
	}
}

func (s synchronizer) process(ctx context.Context, e event.Event, logger *log.Entry) error {
	if !e.IsCreated() {
		logger.Debugf("ignoring event '%s'", e)
		return nil
	}

	logger.Infof("processing event '%s' for '%s'...", e, e.Application)

	var apps v1.AzureAdApplicationList
	err := s.kubeReader.List(ctx, &apps)
	if err != nil {
		return fmt.Errorf("fetching AzureAdApplications from cluster: %w", err)
	}

	candidateCount := 0

	for _, app := range apps.Items {
		app.SetClusterName(s.config.ClusterName)

		if customresources.ShouldResynchronize(app, e) {
			candidateID := kubernetes.UniformResourceName(&app)
			candidateCount += 1

			if err := s.resync(ctx, app); err != nil {
				return fmt.Errorf("resyncing %s: %w", candidateID, err)
			}

			logger.Infof("marked '%s' for resync", candidateID)
		}
	}

	logger.Infof("found and marked %d candidates for resync", candidateCount)
	return nil
}

func (s synchronizer) resync(ctx context.Context, app v1.AzureAdApplication) error {
	existing := &v1.AzureAdApplication{}
	key := client.ObjectKey{Namespace: app.Namespace, Name: app.Name}

	if err := s.kubeReader.Get(ctx, key, existing); err != nil {
		return fmt.Errorf("getting newest version from cluster: %s", err)
	}

	annotations.SetAnnotation(existing, annotations.ResynchronizeKey, strconv.FormatBool(true))

	if err := s.kubeClient.Update(ctx, existing); err != nil {
		return fmt.Errorf("setting resync annotation: %w", err)
	}

	return nil
}
