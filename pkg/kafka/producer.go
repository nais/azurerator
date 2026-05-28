package kafka

import (
	"crypto/tls"
	"os"
	"time"

	"github.com/IBM/sarama"

	"github.com/nais/azureator/pkg/config"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer(config config.Config, tlsConfig *tls.Config) (*Producer, error) {
	cfg := sarama.NewConfig()
	cfg.Net.TLS.Enable = true
	cfg.Net.TLS.Config = tlsConfig
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Errors = true
	cfg.Producer.Return.Successes = true
	cfg.ClientID, _ = os.Hostname()

	syncProducer, err := sarama.NewSyncProducer(config.Kafka.Brokers, cfg)
	if err != nil {
		return nil, err
	}

	return &Producer{
		producer: syncProducer,
		topic:    config.Kafka.Topic,
	}, nil
}

func (p *Producer) Send(message []byte) (int64, error) {
	producerMessage := &sarama.ProducerMessage{
		Topic:     p.topic,
		Value:     sarama.ByteEncoder(message),
		Timestamp: time.Now(),
	}
	_, offset, err := p.producer.SendMessage(producerMessage)
	return offset, err
}
