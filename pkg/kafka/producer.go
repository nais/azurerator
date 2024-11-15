package kafka

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"github.com/IBM/sarama"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/event"
)

type Producer interface {
	Produce(msg Message) (int64, error)
	ProduceEvent(event.Event) (int64, error)
}

type producer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer(config config.Config, tlsConfig *tls.Config, logger *log.Logger) (Producer, error) {
	cfg := sarama.NewConfig()
	cfg.Net.TLS.Enable = true
	cfg.Net.TLS.Config = tlsConfig
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Errors = true
	cfg.Producer.Return.Successes = true
	cfg.ClientID, _ = os.Hostname()
	sarama.Logger = logger

	syncProducer, err := sarama.NewSyncProducer(config.Kafka.Brokers, cfg)
	if err != nil {
		return nil, err
	}

	return &producer{
		producer: syncProducer,
		topic:    config.Kafka.Topic,
	}, nil
}

func (p *producer) Produce(msg Message) (offset int64, err error) {
	producerMessage := &sarama.ProducerMessage{
		Topic:     p.topic,
		Value:     sarama.ByteEncoder(msg),
		Timestamp: time.Now(),
	}
	_, offset, err = p.producer.SendMessage(producerMessage)
	return
}

func (p *producer) ProduceEvent(e event.Event) (int64, error) {
	message, err := e.Marshal()
	if err != nil {
		return -1, fmt.Errorf("marshalling event: %w", err)
	}

	return p.Produce(message)
}
