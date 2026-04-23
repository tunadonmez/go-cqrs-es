package infrastructure

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	commonevents "github.com/tunadonmez/go-cqrs-es/wallet-common/events"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/observability"
)

const eventsTopic = "WalletEvents"

// kafkaEnvelope matches the envelope produced by the write side.
// EventID is surfaced on the envelope for observability and also lives
// inside the serialized event payload itself (via BaseEventData.EventID).
type kafkaEnvelope struct {
	EventID string          `json:"eventId"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
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

	slog.Info("Kafka consumer started", "component", "kafka-consumer", "topic", topic, "groupId", c.groupID)
	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // context cancelled
			}
			observability.DefaultMetrics.KafkaMessageFailures.Add(1)
			slog.Error("Kafka read failed", "component", "kafka-consumer", "topic", topic, "groupId", c.groupID, "error", err)
			continue
		}

		var envelope kafkaEnvelope
		if err := json.Unmarshal(m.Value, &envelope); err != nil {
			observability.DefaultMetrics.KafkaMessageFailures.Add(1)
			slog.Error("Kafka envelope decode failed",
				"component", "kafka-consumer",
				"topic", topic,
				"groupId", c.groupID,
				"partition", m.Partition,
				"offset", m.Offset,
				"error", err)
			continue
		}

		observability.DefaultMetrics.KafkaMessagesConsumed.Add(1)
		slog.Info("Kafka message consumed",
			"component", "kafka-consumer",
			"topic", topic,
			"groupId", c.groupID,
			"partition", m.Partition,
			"offset", m.Offset,
			"eventId", envelope.EventID,
			"eventType", envelope.Type)

		if err := c.dispatch(envelope); err != nil {
			// Leave the event unprocessed; Kafka will redeliver on restart or
			// rebalance. The inbox makes duplicate deliveries safe.
			observability.DefaultMetrics.KafkaMessageFailures.Add(1)
			slog.Error("Kafka message handling failed",
				"component", "kafka-consumer",
				"topic", topic,
				"groupId", c.groupID,
				"eventId", envelope.EventID,
				"eventType", envelope.Type,
				"error", err)
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
		fallbackEventID(&event.BaseEventData.EventID, envelope.EventID)
		return c.eventHandler.OnWalletCreated(&event)

	case "WalletCreditedEvent":
		var event commonevents.WalletCreditedEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return err
		}
		fallbackEventID(&event.BaseEventData.EventID, envelope.EventID)
		return c.eventHandler.OnWalletCredited(&event)

	case "WalletDebitedEvent":
		var event commonevents.WalletDebitedEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return err
		}
		fallbackEventID(&event.BaseEventData.EventID, envelope.EventID)
		return c.eventHandler.OnWalletDebited(&event)

	default:
		observability.DefaultMetrics.KafkaMessageFailures.Add(1)
		slog.Warn("Kafka event type is unknown", "component", "kafka-consumer", "eventType", envelope.Type, "eventId", envelope.EventID)
		return nil
	}
}

// fallbackEventID copies the envelope-level id onto the inner event when the
// serialized payload did not carry one (older producers).
func fallbackEventID(target *string, envelopeID string) {
	if *target == "" {
		*target = envelopeID
	}
}
