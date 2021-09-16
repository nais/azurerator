package synchronizer

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Shopify/sarama"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
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
	kubeClient client.Client
	kubeReader client.Reader
	config     config.Config
	stopChan   chan struct{}
}

func NewSynchronizer(
	kubeClient client.Client,
	kubeReader client.Reader,
	config config.Config,
) Synchronizer {
	stopChan := make(chan struct{}, 1)

	return synchronizer{
		kubeClient: kubeClient,
		kubeReader: kubeReader,
		config:     config,
		stopChan:   stopChan,
	}
}

func (s synchronizer) Callback() kafka.Callback {
	return func(msg *sarama.ConsumerMessage, logger *log.Entry) (bool, error) {
		logger.Debugf("incoming message from Kafka")

		eventMsg := &event.Event{}
		err := json.Unmarshal(msg.Value, &eventMsg)
		if err != nil {
			return false, fmt.Errorf("unmarshalling message to event; ignoring: %w", err)
		}

		logger = logger.WithFields(log.Fields{
			"CorrelationID": eventMsg.ID,
			"application":   eventMsg.Application,
			"event_name":    eventMsg.EventName,
		})

		err = s.processEvent(context.Background(), *eventMsg, logger)
		if err != nil {
			return true, fmt.Errorf("processing event: %w", err)
		}

		logger.Info("event synchronized")
		return false, nil
	}
}

func (s synchronizer) processEvent(ctx context.Context, e event.Event, logger *log.Entry) error {
	if e.EventName != event.Created {
		logger.Debugf("ignoring event '%s'", e.EventName)
		return nil
	}

	logger.Infof("processing event '%s' for '%s:%s:%s'...", e.EventName, e.Application.Cluster, e.Application.Namespace, e.Application.Name)

	candidates, err := s.findResyncCandidates(ctx, e)
	if err != nil {
		return err
	}

	return s.resyncAll(ctx, candidates, logger)
}

func (s synchronizer) findResyncCandidates(ctx context.Context, e event.Event) ([]v1.AzureAdApplication, error) {
	candidates := make([]v1.AzureAdApplication, 0)
	apps, err := s.fetchAzureAdApplicationsFromCluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching AzureAdApplications from cluster: %w", err)
	}

	for _, app := range apps {
		if customresources.ShouldResynchronize(app, e) {
			candidates = append(candidates, app)
		}
	}

	return candidates, nil
}

func (s synchronizer) fetchAzureAdApplicationsFromCluster(ctx context.Context) ([]v1.AzureAdApplication, error) {
	var apps v1.AzureAdApplicationList

	err := s.kubeReader.List(ctx, &apps)
	if err != nil {
		return nil, err
	}

	return apps.Items, nil
}

func (s synchronizer) resyncAll(ctx context.Context, candidates []v1.AzureAdApplication, logger *log.Entry) error {
	candidateCount := len(candidates)
	logger.Debugf("found %d candidates to resync", candidateCount)

	for i, candidate := range candidates {
		candidateID := fmt.Sprintf("%s:%s:%s", s.config.ClusterName, candidate.GetNamespace(), candidate.GetName())

		err := s.resync(ctx, candidate)
		if err != nil {
			return fmt.Errorf("resyncing %s: %w", candidateID, err)
		}

		logger.Infof("[%d/%d] marked '%s' for resync", i+1, candidateCount, candidateID)
	}

	return nil
}

func (s synchronizer) resync(ctx context.Context, app v1.AzureAdApplication) error {
	existing := &v1.AzureAdApplication{}

	err := s.kubeReader.Get(ctx, client.ObjectKey{Namespace: app.Namespace, Name: app.Name}, existing)
	if err != nil {
		return fmt.Errorf("getting newest version from cluster: %s", err)
	}

	annotations.SetAnnotation(existing, annotations.ResynchronizeKey, strconv.FormatBool(true))

	if err := s.kubeClient.Update(ctx, existing); err != nil {
		return fmt.Errorf("setting resync annotation: %w", err)
	}

	return nil
}
