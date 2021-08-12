package kafka

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/event"
)

type Consumer interface {
	Consume(context.Context) chan EventMessage
	CommitRead(context.Context, ...kafka.Message) error
	Close() error
}

type consumer struct {
	reader *kafka.Reader
}

func NewConsumer(clientID string, config config.Config, tlsConfig *tls.Config) Consumer {
	reader := kafkaReader(clientID, config, tlsConfig)
	return consumer{reader: reader}
}

func (c consumer) Consume(ctx context.Context) chan EventMessage {
	messages := make(chan EventMessage)

	go func(ctx context.Context, messages chan EventMessage) {
		defer close(messages)
		cctx, cancel := context.WithCancel(ctx)
		defer cancel()

		for {
			msg, err := c.reader.FetchMessage(cctx)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					log.Errorf("fetching message from kafka: %+v", err)
				}
				return
			}

			eventMsg := &event.Event{}
			err = json.Unmarshal(msg.Value, &eventMsg)
			if err != nil {
				log.Errorf("unmarshalling message to event; ignoring: %+v", err)

				err = c.CommitRead(ctx, msg)
				if err != nil {
					log.Error(err)
					return
				}

				continue
			}

			messages <- NewEventMessage(*eventMsg, msg)
		}
	}(ctx, messages)

	return messages
}

func (c consumer) CommitRead(ctx context.Context, messages ...kafka.Message) error {
	err := c.reader.CommitMessages(ctx, messages...)
	if err != nil {
		return fmt.Errorf("commiting offset to kafka: %w", err)
	}

	return nil
}

func (c consumer) Close() error {
	return c.reader.Close()
}

func kafkaReader(clientID string, config config.Config, tlsConfig *tls.Config) *kafka.Reader {
	dialer := &kafka.Dialer{
		ClientID:  clientID,
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	if config.Kafka.TLS.Enabled {
		dialer.TLS = tlsConfig
	}

	groupID := fmt.Sprintf("azurerator-%s-%s-v1", config.ClusterName, config.Azure.Tenant.Id)

	readerCfg := kafka.ReaderConfig{
		Brokers:               config.Kafka.Brokers,
		GroupID:               groupID,
		Topic:                 config.Kafka.Topic,
		Dialer:                dialer,
		StartOffset:           kafka.LastOffset,
		WatchPartitionChanges: true,
	}
	return kafka.NewReader(readerCfg)
}
