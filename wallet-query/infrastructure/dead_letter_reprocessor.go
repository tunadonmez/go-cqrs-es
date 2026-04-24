package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/tunadonmez/go-cqrs-es/wallet-common/observability"
)

// DeadLetterReprocessor replays one dead-letter row through the same
// envelope dispatch path the Kafka consumer uses.
type DeadLetterReprocessor struct {
	deadLetters  *DeadLetterRepository
	eventHandler *WalletEventHandler
}

func NewDeadLetterReprocessor(deadLetters *DeadLetterRepository, eventHandler *WalletEventHandler) *DeadLetterReprocessor {
	return &DeadLetterReprocessor{
		deadLetters:  deadLetters,
		eventHandler: eventHandler,
	}
}

func (r *DeadLetterReprocessor) Reprocess(ctx context.Context, deadLetterKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	observability.DefaultMetrics.DeadLetterReprocessRuns.Add(1)

	record, err := r.deadLetters.FindByKey(deadLetterKey)
	if err != nil {
		observability.DefaultMetrics.DeadLetterReprocessFailures.Add(1)
		return fmt.Errorf("load dead letter %q: %w", deadLetterKey, err)
	}

	slog.Info("Dead-letter reprocessing started",
		"component", "dead-letter-reprocess",
		"deadLetterKey", record.DeadLetterKey,
		"eventId", record.EventID,
		"eventType", record.EventType,
		"aggregateId", record.AggregateID,
		"status", record.Status)

	var envelope EventEnvelope
	if err := json.Unmarshal([]byte(record.Payload), &envelope); err != nil {
		updateErr := r.deadLetters.MarkFailedReprocess(record.DeadLetterKey, err, nowUTC())
		observability.DefaultMetrics.DeadLetterReprocessFailures.Add(1)
		if updateErr != nil {
			return fmt.Errorf("decode dead-letter payload: %w (also failed to update dead-letter record: %v)", err, updateErr)
		}
		return fmt.Errorf("decode dead-letter payload: %w", err)
	}

	if envelope.EventID == "" {
		envelope.EventID = record.EventID
	}
	aggregateID := aggregateIDFromEnvelope(envelope)
	if aggregateID == "" {
		aggregateID = record.AggregateID
	}

	if err := DispatchEnvelope(r.eventHandler, envelope); err != nil {
		now := nowUTC()
		updateErr := r.deadLetters.MarkFailedReprocess(record.DeadLetterKey, err, now)
		observability.DefaultMetrics.DeadLetterReprocessFailures.Add(1)
		slog.Error("Dead-letter reprocessing failed",
			"component", "dead-letter-reprocess",
			"deadLetterKey", record.DeadLetterKey,
			"eventId", envelope.EventID,
			"eventType", envelope.Type,
			"aggregateId", aggregateID,
			"error", err)
		if updateErr != nil {
			return fmt.Errorf("reprocess dead letter %q: %w (also failed to update dead-letter record: %v)", record.DeadLetterKey, err, updateErr)
		}
		return fmt.Errorf("reprocess dead letter %q: %w", record.DeadLetterKey, err)
	}

	now := nowUTC()
	if err := r.deadLetters.MarkResolved(record.DeadLetterKey, now); err != nil {
		observability.DefaultMetrics.DeadLetterReprocessFailures.Add(1)
		return fmt.Errorf("mark dead letter %q resolved: %w", record.DeadLetterKey, err)
	}

	observability.DefaultMetrics.DeadLetterReprocessed.Add(1)
	slog.Info("Dead-letter reprocessing succeeded",
		"component", "dead-letter-reprocess",
		"deadLetterKey", record.DeadLetterKey,
		"eventId", envelope.EventID,
		"eventType", envelope.Type,
		"aggregateId", aggregateID)

	return nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
