package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/observability"
)

const (
	eventsTopic            = "WalletEvents"
	maxProjectionAttempts  = 3
	retryBackoff           = 500 * time.Millisecond
	commitAttempts         = 3
	deadLetterSaveAttempts = 3
	consumerRestartBackoff = 2 * time.Second
)

var errPermanentConsumerFailure = errors.New("permanent consumer failure")

type consumedMessage struct {
	topic      string
	partition  int
	offset     int64
	groupID    string
	rawPayload []byte
	envelope   EventEnvelope
}

// WalletEventConsumer consumes domain events from Kafka and projects them.
type WalletEventConsumer struct {
	bootstrapServers string
	groupID          string
	eventHandler     *WalletEventHandler
	deadLetters      *DeadLetterRepository
}

func NewWalletEventConsumer(
	bootstrap, groupID string,
	handler *WalletEventHandler,
	deadLetters *DeadLetterRepository,
) *WalletEventConsumer {
	return &WalletEventConsumer{
		bootstrapServers: bootstrap,
		groupID:          groupID,
		eventHandler:     handler,
		deadLetters:      deadLetters,
	}
}

// Start launches a single consumer for all wallet events, ensuring
// per-aggregate ordering via partition key.
func (c *WalletEventConsumer) Start(ctx context.Context) {
	go c.run(ctx, eventsTopic)
}

func (c *WalletEventConsumer) run(ctx context.Context, topic string) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := c.consume(ctx, topic); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Kafka consumer loop stopped",
				"component", "kafka-consumer",
				"topic", topic,
				"groupId", c.groupID,
				"error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(consumerRestartBackoff):
			}
		}
	}
}

func (c *WalletEventConsumer) consume(ctx context.Context, topic string) error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{c.bootstrapServers},
		Topic:          topic,
		GroupID:        c.groupID,
		CommitInterval: 0,
	})
	defer r.Close()

	slog.Info("Kafka consumer started", "component", "kafka-consumer", "topic", topic, "groupId", c.groupID)
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			observability.DefaultMetrics.KafkaMessageFailures.Add(1)
			return fmt.Errorf("fetch kafka message: %w", err)
		}

		msg := consumedMessage{
			topic:      topic,
			partition:  m.Partition,
			offset:     m.Offset,
			groupID:    c.groupID,
			rawPayload: m.Value,
		}

		if err := json.Unmarshal(m.Value, &msg.envelope); err != nil {
			observability.DefaultMetrics.KafkaMessageFailures.Add(1)
			slog.Error("Kafka envelope decode failed",
				"component", "kafka-consumer",
				"topic", topic,
				"groupId", c.groupID,
				"partition", m.Partition,
				"offset", m.Offset,
				"error", err)
			if err := c.deadLetterAndCommit(ctx, r, m, msg, "permanent", 1, fmt.Errorf("%w: decode envelope: %v", errPermanentConsumerFailure, err)); err != nil {
				return err
			}
			continue
		}

		observability.DefaultMetrics.KafkaMessagesConsumed.Add(1)
		slog.Info("Kafka message consumed",
			"component", "kafka-consumer",
			"topic", topic,
			"groupId", c.groupID,
			"partition", m.Partition,
			"offset", m.Offset,
			"eventId", msg.envelope.EventID,
			"eventType", msg.envelope.Type)

		if err := c.processMessage(ctx, r, m, msg); err != nil {
			return err
		}
	}
}

func (c *WalletEventConsumer) processMessage(
	ctx context.Context,
	r *kafka.Reader,
	m kafka.Message,
	msg consumedMessage,
) error {
	lastErr := error(nil)
	for attempt := 1; attempt <= maxProjectionAttempts; attempt++ {
		err := c.dispatch(msg.envelope)
		if err == nil {
			if err := c.commitMessage(ctx, r, m, msg); err != nil {
				return err
			}
			return nil
		}

		lastErr = err
		observability.DefaultMetrics.KafkaMessageFailures.Add(1)
		if errors.Is(err, errPermanentConsumerFailure) {
			slog.Error("Kafka message handling failed permanently",
				"component", "kafka-consumer",
				"topic", msg.topic,
				"groupId", msg.groupID,
				"partition", msg.partition,
				"offset", msg.offset,
				"eventId", msg.envelope.EventID,
				"eventType", msg.envelope.Type,
				"error", err)
			return c.deadLetterAndCommit(ctx, r, m, msg, "permanent", attempt, err)
		}

		if attempt < maxProjectionAttempts {
			observability.DefaultMetrics.KafkaRetryAttempts.Add(1)
			slog.Warn("Kafka projection retry scheduled",
				"component", "kafka-consumer",
				"topic", msg.topic,
				"groupId", msg.groupID,
				"partition", msg.partition,
				"offset", msg.offset,
				"eventId", msg.envelope.EventID,
				"eventType", msg.envelope.Type,
				"attempt", attempt,
				"maxAttempts", maxProjectionAttempts,
				"backoff", retryBackoff,
				"error", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(retryBackoff):
			}
			continue
		}
	}

	slog.Error("Kafka message handling exhausted retries",
		"component", "kafka-consumer",
		"topic", msg.topic,
		"groupId", msg.groupID,
		"partition", msg.partition,
		"offset", msg.offset,
		"eventId", msg.envelope.EventID,
		"eventType", msg.envelope.Type,
		"attempts", maxProjectionAttempts,
		"error", lastErr)
	return c.deadLetterAndCommit(ctx, r, m, msg, "retries_exhausted", maxProjectionAttempts, lastErr)
}

func (c *WalletEventConsumer) deadLetterAndCommit(
	ctx context.Context,
	r *kafka.Reader,
	m kafka.Message,
	msg consumedMessage,
	failureKind string,
	attempts int,
	failure error,
) error {
	if err := c.saveDeadLetter(msg, failureKind, attempts, failure); err != nil {
		return err
	}
	return c.commitMessage(ctx, r, m, msg)
}

func (c *WalletEventConsumer) saveDeadLetter(
	msg consumedMessage,
	failureKind string,
	attempts int,
	failure error,
) error {
	now := time.Now().UTC()
	record := &DeadLetterEvent{
		DeadLetterKey:      deadLetterKey(msg),
		EventID:            msg.envelope.EventID,
		EventType:          msg.envelope.Type,
		EventSchemaVersion: normalizedEnvelopeSchemaVersion(msg.envelope),
		AggregateID:        aggregateIDFromEnvelope(msg.envelope),
		Topic:              msg.topic,
		Partition:          msg.partition,
		Offset:             msg.offset,
		ConsumerGroup:      msg.groupID,
		FailureKind:        failureKind,
		RetryAttempts:      attempts,
		LastError:          failure.Error(),
		Payload:            string(msg.rawPayload),
		FirstFailedAt:      now,
		LastFailedAt:       now,
		DeadLetteredAt:     now,
	}

	var lastErr error
	for attempt := 1; attempt <= deadLetterSaveAttempts; attempt++ {
		if err := c.deadLetters.Save(record); err != nil {
			lastErr = err
			observability.DefaultMetrics.DeadLetterSaveFailures.Add(1)
			slog.Error("Dead-letter save failed",
				"component", "kafka-consumer",
				"topic", msg.topic,
				"groupId", msg.groupID,
				"partition", msg.partition,
				"offset", msg.offset,
				"eventId", msg.envelope.EventID,
				"eventType", msg.envelope.Type,
				"attempt", attempt,
				"error", err)
			continue
		}

		observability.DefaultMetrics.DeadLetteredEvents.Add(1)
		slog.Error("Kafka message moved to dead letter",
			"component", "kafka-consumer",
			"topic", msg.topic,
			"groupId", msg.groupID,
			"partition", msg.partition,
			"offset", msg.offset,
			"eventId", msg.envelope.EventID,
			"eventType", msg.envelope.Type,
			"aggregateId", record.AggregateID,
			"failureKind", failureKind,
			"retryAttempts", attempts,
			"deadLetterKey", record.DeadLetterKey,
			"error", failure)
		return nil
	}

	return fmt.Errorf("save dead letter after %d attempts: %w", deadLetterSaveAttempts, lastErr)
}

func (c *WalletEventConsumer) commitMessage(
	ctx context.Context,
	r *kafka.Reader,
	m kafka.Message,
	msg consumedMessage,
) error {
	var lastErr error
	for attempt := 1; attempt <= commitAttempts; attempt++ {
		if err := r.CommitMessages(ctx, m); err != nil {
			lastErr = err
			slog.Error("Kafka offset commit failed",
				"component", "kafka-consumer",
				"topic", msg.topic,
				"groupId", msg.groupID,
				"partition", msg.partition,
				"offset", msg.offset,
				"eventId", msg.envelope.EventID,
				"eventType", msg.envelope.Type,
				"attempt", attempt,
				"error", err)
			continue
		}
		return nil
	}
	return fmt.Errorf("commit kafka message after %d attempts: %w", commitAttempts, lastErr)
}

func (c *WalletEventConsumer) dispatch(envelope EventEnvelope) error {
	return DispatchEnvelope(c.eventHandler, envelope)
}

func deadLetterKey(msg consumedMessage) string {
	if msg.envelope.EventID != "" {
		return msg.envelope.EventID
	}
	return fmt.Sprintf("%s:%d:%d", msg.topic, msg.partition, msg.offset)
}
