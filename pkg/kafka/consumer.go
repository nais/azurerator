package kafka

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/IBM/sarama"
	log "github.com/sirupsen/logrus"

	"github.com/nais/azureator/pkg/config"
)

type Callback func(message *sarama.ConsumerMessage, logger *log.Entry) (retry bool, err error)

var _ sarama.ConsumerGroupHandler = (*Consumer)(nil)

type Consumer struct {
	callback      Callback
	logger        *log.Logger
	retryInterval time.Duration
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *Consumer) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *Consumer) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	retry := true
	var err error

	for message := range claim.Messages() {
		for retry {
			logger := c.logger.WithFields(log.Fields{
				"kafka_offset":    message.Offset,
				"kafka_partition": message.Partition,
				"kafka_topic":     message.Topic,
			})
			retry, err = c.callback(message, logger)
			if err != nil {
				logger.Errorf("consuming Kafka message: %s", err)
				if retry {
					time.Sleep(c.retryInterval)
				}
			}
		}
		retry, err = true, nil
		session.MarkMessage(message, "")
	}
	return nil
}

func NewConsumer(ctx context.Context, cfg config.Config, tlsConfig *tls.Config, logger *log.Logger, callback Callback) (*Consumer, error) {
	consumerCfg := sarama.NewConfig()
	consumerCfg.Net.TLS.Enable = cfg.Kafka.TLS.Enabled
	consumerCfg.Net.TLS.Config = tlsConfig
	consumerCfg.Version = sarama.V3_1_0_0
	consumerCfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	consumerCfg.Consumer.MaxProcessingTime = cfg.Kafka.MaxProcessingTime
	consumerCfg.ClientID, _ = os.Hostname()
	sarama.Logger = logger

	groupID := fmt.Sprintf("azurerator-%s-%s-v1", cfg.ClusterName, cfg.Azure.Tenant.Id)

	group, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, groupID, consumerCfg)
	if err != nil {
		return nil, err
	}

	c := &Consumer{
		callback:      callback,
		logger:        logger,
		retryInterval: cfg.Kafka.RetryInterval,
	}

	go func() {
		for err := range group.Errors() {
			c.logger.Errorf("Consumer encountered error: %s", err)
		}
	}()

	go func() {
		for {
			c.logger.Infof("(re-)starting consumer on topic %s", cfg.Kafka.Topic)
			err := group.Consume(ctx, []string{cfg.Kafka.Topic}, c)
			if err != nil {
				c.logger.Errorf("Error consuming: %s", err)
			}

			// check if context was cancelled, signaling that the consumer should stop
			if errors.Is(ctx.Err(), context.Canceled) {
				c.logger.Debug("Consumer context cancelled, stopping consumer")
				return
			}

			time.Sleep(10 * time.Second)
		}
	}()

	return c, nil
}
