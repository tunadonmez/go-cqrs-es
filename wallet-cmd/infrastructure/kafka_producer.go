package infrastructure

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
)

// KafkaMessage is the envelope sent over Kafka. EventID is surfaced at the
// envelope level so consumers can log and dedupe without unmarshalling the
// inner event payload.
type KafkaMessage struct {
	EventID       string          `json:"eventId"`
	SchemaVersion int             `json:"schemaVersion,omitempty"`
	Type          string          `json:"type"`
	Data          json.RawMessage `json:"data"`
}

// WalletEventProducer publishes domain events to Kafka.
type WalletEventProducer struct {
	writer *kafka.Writer
}

func NewWalletEventProducer(bootstrapServers string) *WalletEventProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(bootstrapServers),
		Balancer:     &kafka.Hash{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireAll,
	}
	return &WalletEventProducer{writer: writer}
}

func (p *WalletEventProducer) Close() error {
	return p.writer.Close()
}

func (p *WalletEventProducer) Produce(topic string, event corevents.BaseEvent) error {
	corevents.EnsureSchemaVersion(event)
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	envelope := KafkaMessage{
		EventID:       event.GetEventID(),
		SchemaVersion: event.GetSchemaVersion(),
		Type:          event.EventTypeName(),
		Data:          data,
	}
	msg, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use the aggregate ID as the partition key so all events for the same
	// aggregate land in the same partition, preserving ordering.
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(event.GetAggregateID()),
		Value: msg,
	})
}
