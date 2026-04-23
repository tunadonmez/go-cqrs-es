package infrastructure

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
)

// KafkaMessage is the envelope sent over Kafka.
type KafkaMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// WalletEventProducer publishes domain events to Kafka.
type WalletEventProducer struct {
	bootstrapServers string
}

func NewWalletEventProducer(bootstrapServers string) *WalletEventProducer {
	return &WalletEventProducer{bootstrapServers: bootstrapServers}
}

func (p *WalletEventProducer) Produce(topic string, event corevents.BaseEvent) error {
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
