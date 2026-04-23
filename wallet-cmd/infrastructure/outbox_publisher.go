package infrastructure

import (
	"context"
	"log"
	"time"

	"github.com/tunadonmez/go-cqrs-es/cqrs-core/producers"
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
	log.Printf("Outbox publisher started (topic=%s, interval=%s, batchSize=%d)", p.topic, p.interval, p.batchSize)
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
			log.Printf("outbox: load pending failed: %v", err)
			return
		}
		if len(pending) == 0 {
			return
		}
		for _, entry := range pending {
			if err := p.publishEntry(ctx, entry); err != nil {
				log.Printf("outbox: publish failed (eventId=%s type=%s aggregate=%s): %v",
					entry.EventID, entry.EventType, entry.AggregateIdentifier, err)
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
		if recErr := p.repository.RecordPublishFailure(ctx, entry.ID, time.Now().UTC(), err); recErr != nil {
			log.Printf("outbox: failed to record failure for %s: %v", entry.EventID, recErr)
		}
		return err
	}
	return p.repository.MarkPublished(ctx, entry.ID, time.Now().UTC())
}
