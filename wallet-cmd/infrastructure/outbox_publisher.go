package infrastructure

import (
	"context"
	"log/slog"
	"time"

	"github.com/tunadonmez/go-cqrs-es/cqrs-core/producers"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/observability"
)

// OutboxPublisher drains pending events from the event-store-as-outbox into
// Kafka. It runs a single loop, so events are published in insertion order
// (by `_id`), which — combined with the aggregate-ID partition key — preserves
// per-aggregate ordering on the broker.
type OutboxPublisher struct {
	repository *EventStoreRepository
	producer   producers.EventProducer
	topic      string
	interval   time.Duration
	batchSize  int64
}

// OutboxPublisherOption configures the publisher.
type OutboxPublisherOption func(*OutboxPublisher)

func WithPollInterval(d time.Duration) OutboxPublisherOption {
	return func(p *OutboxPublisher) { p.interval = d }
}

func WithBatchSize(n int64) OutboxPublisherOption {
	return func(p *OutboxPublisher) { p.batchSize = n }
}

func NewOutboxPublisher(repo *EventStoreRepository, producer producers.EventProducer, topic string, opts ...OutboxPublisherOption) *OutboxPublisher {
	p := &OutboxPublisher{
		repository: repo,
		producer:   producer,
		topic:      topic,
		interval:   500 * time.Millisecond,
		batchSize:  100,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Start runs the publisher loop until the context is cancelled.
func (p *OutboxPublisher) Start(ctx context.Context) {
	go p.loop(ctx)
}

func (p *OutboxPublisher) loop(ctx context.Context) {
	slog.Info("Outbox publisher started", "topic", p.topic, "interval", p.interval, "batchSize", p.batchSize)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.drain(ctx)
		}
	}
}

// drain publishes as many batches as are available in one tick. This keeps
// the publisher responsive under bursts without depending on the tick rate.
func (p *OutboxPublisher) drain(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		pending, err := p.repository.FindPendingOutbox(ctx, p.batchSize)
		if err != nil {
			slog.Error("outbox: load pending failed", "error", err)
			return
		}
		if len(pending) == 0 {
			return
		}
		slog.Debug("outbox: processing batch", "count", len(pending))
		for _, entry := range pending {
			if err := p.publishEntry(ctx, entry); err != nil {
				slog.Error("outbox: publish failed",
					"eventId", entry.EventID,
					"type", entry.EventType,
					"aggregateId", entry.AggregateIdentifier,
					"error", err)
				// Stop the batch early: publishing further events for the same
				// aggregate before a retry would violate per-aggregate ordering.
				return
			}
		}
		if int64(len(pending)) < p.batchSize {
			return
		}
	}
}

func (p *OutboxPublisher) publishEntry(ctx context.Context, entry *EventModelDoc) error {
	if err := p.producer.Produce(p.topic, entry.EventData); err != nil {
		observability.DefaultMetrics.ProduceFailures.Add(1)
		if recErr := p.repository.RecordPublishFailure(ctx, entry.ID, time.Now().UTC(), err); recErr != nil {
			slog.Error("outbox: failed to record failure", "eventId", entry.EventID, "error", recErr)
		}
		return err
	}
	if err := p.repository.MarkPublished(ctx, entry.ID, time.Now().UTC()); err != nil {
		slog.Error("outbox: failed to mark published", "eventId", entry.EventID, "error", err)
		return err
	}
	observability.DefaultMetrics.ProducedEvents.Add(1)
	slog.Info("Outbox event published to Kafka",
		"eventId", entry.EventID,
		"type", entry.EventType,
		"aggregateId", entry.AggregateIdentifier)
	return nil
}
