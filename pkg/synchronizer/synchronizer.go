package synchronizer

import (
	"context"
	"fmt"
	"strconv"
	"time"

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
	Synchronize()
	Close() error
}

type synchronizer struct {
	kafkaClient kafka.Client
	kubeClient  client.Client
	kubeReader  client.Reader
	config      config.Config
	logger      *log.Entry
	stopChan    chan struct{}
}

func NewSynchronizer(
	kafkaClient kafka.Client,
	kubeClient client.Client,
	kubeReader client.Reader,
	config config.Config,
) Synchronizer {
	stopChan := make(chan struct{}, 1)

	return synchronizer{
		kafkaClient: kafkaClient,
		kubeClient:  kubeClient,
		kubeReader:  kubeReader,
		config:      config,
		stopChan:    stopChan,
	}
}

func (s synchronizer) Synchronize() {
	ctx := context.Background()
	messages := s.kafkaClient.Consume(ctx)

	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				log.Errorf("lost connection to kafka; retrying...")
				time.Sleep(5 * time.Second)
				messages = s.kafkaClient.Consume(ctx)
				continue
			}

			cctx, cancel := context.WithTimeout(ctx, 5*time.Second)

			err := s.sync(cctx, msg)
			cancel()
			if err != nil {
				s.logger.Errorf("synchronizing event: %v", err)
				time.Sleep(10 * time.Second)
				s.logger.Debugf("retrying...")
				messages <- msg
			}
		case <-s.stopChan:
			return
		}
	}
}

func (s synchronizer) Close() error {
	close(s.stopChan)
	return s.kafkaClient.Close()
}

func (s synchronizer) sync(ctx context.Context, msg kafka.EventMessage) error {
	s.logger = log.WithFields(log.Fields{
		"CorrelationID": msg.ID,
		"application":   msg.Application,
		"event_name":    msg.EventName,
		"kafka":         msg.Metadata(),
	})

	err := s.processEvent(ctx, msg.Event)
	if err != nil {
		return fmt.Errorf("processing event: %w", err)
	}

	err = s.kafkaClient.CommitRead(ctx, msg.Message)
	if err != nil {
		return err
	}

	s.logger.Info("event synchronized.")
	return nil
}

func (s synchronizer) processEvent(ctx context.Context, e event.Event) error {
	if e.EventName != event.Created {
		s.logger.Debugf("ignoring event '%s'", e.EventName)
		return nil
	}

	s.logger.Infof("processing event '%s' for '%s:%s:%s'...", e.EventName, e.Application.Cluster, e.Application.Namespace, e.Application.Name)

	candidates, err := s.findResyncCandidates(ctx, e)
	if err != nil {
		return err
	}

	return s.resyncAll(ctx, candidates)
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

func (s synchronizer) resyncAll(ctx context.Context, candidates []v1.AzureAdApplication) error {
	candidateCount := len(candidates)
	s.logger.Debugf("found %d candidates to resync", candidateCount)

	for i, candidate := range candidates {
		candidateID := fmt.Sprintf("%s:%s:%s", s.config.ClusterName, candidate.GetNamespace(), candidate.GetName())

		err := s.resync(ctx, candidate)
		if err != nil {
			return fmt.Errorf("resyncing %s: %w", candidateID, err)
		}

		s.logger.Infof("[%d/%d] marked '%s' for resync", i+1, candidateCount, candidateID)
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
