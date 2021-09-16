package kafka

import (
	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

type Message []byte

type Callback func(message *sarama.ConsumerMessage, logger *log.Entry) (retry bool, err error)
