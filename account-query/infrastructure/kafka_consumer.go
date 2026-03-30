package infrastructure

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
	commonevents "github.com/techbank/account-common/events"
)

const eventsTopic = "BankAccountEvents"

// kafkaEnvelope matches the envelope produced by the command service.
type kafkaEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// AccountEventConsumer consumes domain events from Kafka and projects them.
type AccountEventConsumer struct {
	bootstrapServers string
	groupID          string
	eventHandler     *AccountEventHandler
}

func NewAccountEventConsumer(bootstrap, groupID string, handler *AccountEventHandler) *AccountEventConsumer {
	return &AccountEventConsumer{
		bootstrapServers: bootstrap,
		groupID:          groupID,
		eventHandler:     handler,
	}
}

// Start launches a single consumer for all account events, ensuring
// per-aggregate ordering via partition key.
func (c *AccountEventConsumer) Start(ctx context.Context) {
	go c.consume(ctx, eventsTopic)
}

func (c *AccountEventConsumer) consume(ctx context.Context, topic string) {
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

func (c *AccountEventConsumer) dispatch(envelope kafkaEnvelope) error {
	switch envelope.Type {
	case "AccountOpenedEvent":
		var event commonevents.AccountOpenedEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return err
		}
		return c.eventHandler.OnAccountOpened(&event)

	case "FundsDepositedEvent":
		var event commonevents.FundsDepositedEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return err
		}
		return c.eventHandler.OnFundsDeposited(&event)

	case "FundsWithdrawnEvent":
		var event commonevents.FundsWithdrawnEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return err
		}
		return c.eventHandler.OnFundsWithdrawn(&event)

	case "AccountClosedEvent":
		var event commonevents.AccountClosedEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return err
		}
		return c.eventHandler.OnAccountClosed(&event)

	default:
		log.Printf("Unknown event type: %s", envelope.Type)
		return nil
	}
}
