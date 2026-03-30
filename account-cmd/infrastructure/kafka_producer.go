package infrastructure

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	corevents "github.com/techbank/cqrs-core/events"
)

// KafkaMessage is the envelope sent over Kafka.
type KafkaMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// AccountEventProducer publishes domain events to Kafka.
type AccountEventProducer struct {
	bootstrapServers string
}

func NewAccountEventProducer(bootstrapServers string) *AccountEventProducer {
	return &AccountEventProducer{bootstrapServers: bootstrapServers}
}

func (p *AccountEventProducer) Produce(topic string, event corevents.BaseEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	envelope := KafkaMessage{Type: event.EventTypeName(), Data: data}
	msg, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{p.bootstrapServers},
		Topic:   topic,
	})
	defer writer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use aggregate ID as the partition key so all events for the same
	// aggregate land in the same partition, preserving ordering.
	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.GetID()),
		Value: msg,
	})
}
