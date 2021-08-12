package kafka

import (
	"github.com/segmentio/kafka-go"

	"github.com/nais/azureator/pkg/event"
)

type EventMessage struct {
	event.Event
	kafka.Message
}

func (in EventMessage) Metadata() Metadata {
	return Metadata{
		Key:       string(in.Key),
		Topic:     in.Topic,
		Partition: in.Partition,
		Offset:    in.Offset,
	}
}

type Metadata struct {
	Key       string `json:"key,omitempty"`
	Topic     string `json:"topic,omitempty"`
	Offset    int64  `json:"offset,omitempty"`
	Partition int    `json:"partition,omitempty"`
}

func NewEventMessage(event event.Event, message kafka.Message) EventMessage {
	return EventMessage{
		Event:   event,
		Message: message,
	}
}
