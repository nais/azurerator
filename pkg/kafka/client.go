package kafka

import (
	"crypto/tls"

	"github.com/google/uuid"

	"github.com/nais/azureator/pkg/config"
)

type Client interface {
	Producer
	Consumer

	Close() error
}

type client struct {
	Producer
	Consumer
}

func NewClient(config config.Config, tlsConfig *tls.Config) Client {
	clientID := uuid.New().String()
	producer := NewProducer(clientID, config, tlsConfig)
	consumer := NewConsumer(clientID, config, tlsConfig)

	return client{
		Producer: producer,
		Consumer: consumer,
	}
}

func (c client) Close() error {
	err := c.Producer.Close()
	if err != nil {
		return err
	}

	err = c.Consumer.Close()
	if err != nil {
		return err
	}

	return nil
}
