package infrastructure

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
	commonevents "github.com/tunadonmez/go-cqrs-es/wallet-common/events"
)

const eventsTopic = "WalletEvents"

// kafkaEnvelope matches the envelope produced by the command service.
type kafkaEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// WalletEventConsumer consumes domain events from Kafka and projects them.
type WalletEventConsumer struct {
	bootstrapServers string
	groupID          string
	eventHandler     *WalletEventHandler
}

func NewWalletEventConsumer(bootstrap, groupID string, handler *WalletEventHandler) *WalletEventConsumer {
	return &WalletEventConsumer{
		bootstrapServers: bootstrap,
		groupID:          groupID,
		eventHandler:     handler,
	}
}

// Start launches a single consumer for all wallet events, ensuring
// per-aggregate ordering via partition key.
func (c *WalletEventConsumer) Start(ctx context.Context) {
	go c.consume(ctx, eventsTopic)
}

func (c *WalletEventConsumer) consume(ctx context.Context, topic string) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{c.bootstrapServers},
		Topic:   topic,
		GroupID: c.groupID,
	})
	defer r.Close()

	log.Printf("Kafka consumer started for topic: %s", topic)
	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // context cancelled
			}
			log.Printf("Error reading from topic %s: %v", topic, err)
			continue
		}

		var envelope kafkaEnvelope
		if err := json.Unmarshal(m.Value, &envelope); err != nil {
			log.Printf("Failed to unmarshal envelope on topic %s: %v", topic, err)
			continue
		}

		if err := c.dispatch(envelope); err != nil {
			log.Printf("Error handling event %s: %v", envelope.Type, err)
		}
	}
}

func (c *WalletEventConsumer) dispatch(envelope kafkaEnvelope) error {
	switch envelope.Type {
	case "WalletCreatedEvent":
		var event commonevents.WalletCreatedEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return err
		}
		return c.eventHandler.OnWalletCreated(&event)

	case "WalletCreditedEvent":
		var event commonevents.WalletCreditedEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return err
		}
		return c.eventHandler.OnWalletCredited(&event)

	case "WalletDebitedEvent":
		var event commonevents.WalletDebitedEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return err
		}
		return c.eventHandler.OnWalletDebited(&event)

	default:
		log.Printf("Unknown event type: %s", envelope.Type)
		return nil
	}
}
