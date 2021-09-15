package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/event"
)

type Producer interface {
	Produce(context.Context, kafka.Message) error
	ProduceEvent(context.Context, event.Event) error
	Close() error
}

type producer struct {
	writer *kafka.Writer
}

func NewProducer(clientID string, config config.Config, tlsConfig *tls.Config) Producer {
	writer := kafkaWriter(clientID, config, tlsConfig)
	return producer{writer: writer}
}

func (p producer) Produce(ctx context.Context, msg kafka.Message) error {
	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		return fmt.Errorf("writing message to kafka: %w", err)
	}

	return nil
}

func (p producer) ProduceEvent(ctx context.Context, eventMsg event.Event) error {
	key := []byte(eventMsg.ID)

	value, err := eventMsg.Marshal()
	if err != nil {
		return fmt.Errorf("marshalling event: %w", err)
	}

	kafkaMsg := kafka.Message{Key: key, Value: value}

	return p.Produce(ctx, kafkaMsg)
}

func (p producer) Close() error {
	return p.writer.Close()
}

func kafkaWriter(clientID string, config config.Config, tlsConfig *tls.Config) *kafka.Writer {
	transport := &kafka.Transport{
		ClientID:    clientID,
		DialTimeout: 1 * time.Minute,
		IdleTimeout: 15 * time.Minute,
	}

	if config.Kafka.TLS.Enabled {
		transport.TLS = tlsConfig
	}

	return &kafka.Writer{
		Addr:         kafka.TCP(config.Kafka.Brokers...),
		Topic:        config.Kafka.Topic,
		RequiredAcks: kafka.RequireAll,
		Transport:    transport,
		BatchSize:    1,
	}
}
