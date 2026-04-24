package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	commonevents "github.com/tunadonmez/go-cqrs-es/wallet-common/events"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/observability"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/domain"
	"gorm.io/gorm"
)

// Replayer rebuilds the PostgreSQL read model from the MongoDB event store.
//
// Replay deliberately bypasses Kafka: it streams events directly from the
// source of truth and hands each one to the same WalletEventHandler methods
// that the live Kafka consumer uses. There is only one projection path —
// applyIdempotent — and both live consumption and replay go through it.
type Replayer struct {
	reader            *EventSourceReader
	eventHandler      *WalletEventHandler
	projectionManager *ProjectionVersionManager
	db                *gorm.DB
}

func NewReplayer(reader *EventSourceReader, handler *WalletEventHandler, projectionManager *ProjectionVersionManager, db *gorm.DB) *Replayer {
	return &Replayer{
		reader:            reader,
		eventHandler:      handler,
		projectionManager: projectionManager,
		db:                db,
	}
}

// ReplayOptions selects which slice of the event store to replay.
type ReplayOptions struct {
	// AggregateID, if non-empty, scopes the replay to a single aggregate.
	// When empty, every event in the store is replayed.
	AggregateID string
}

// Run resets the read model (scoped to the replay's aggregate filter) and
// then streams events from the event store through the projection handlers.
//
// Idempotency: the reset clears both the read-model tables and the
// processed_events inbox so each projection call does real work. Running
// Run a second time is safe — the reset happens first.
func (r *Replayer) Run(ctx context.Context, opts ReplayOptions) error {
	scope := "ALL"
	if opts.AggregateID != "" {
		scope = "aggregate=" + opts.AggregateID
	}

	total, err := r.reader.CountEvents(ctx, opts.AggregateID)
	if err != nil {
		return fmt.Errorf("count events: %w", err)
	}

	observability.DefaultMetrics.ReplayRuns.Add(1)
	slog.Info("Replay started", "component", "replay", "scope", scope, "totalEvents", total)
	start := time.Now().UTC()

	if err := r.resetReadModel(opts.AggregateID); err != nil {
		return fmt.Errorf("reset read model: %w", err)
	}
	slog.Info("Replay read model reset", "component", "replay", "scope", scope)

	var processed int64
	err = r.reader.StreamEvents(ctx, opts.AggregateID, func(ev ReplayEvent) error {
		if err := r.dispatch(ev); err != nil {
			return fmt.Errorf("project %s (aggregate=%s version=%d eventId=%s): %w",
				ev.EventType, ev.AggregateID, ev.Version, ev.EventID, err)
		}
		processed++
		observability.DefaultMetrics.ReplayEventsProcessed.Add(1)
		// Lightweight progress pulse — every 500 events or whenever we hit
		// the total, whichever comes first.
		if processed%500 == 0 || processed == total {
			progress := 0.0
			if total > 0 {
				progress = float64(processed) / float64(total) * 100
			}
			slog.Info("Replay progress", "component", "replay", "scope", scope, "processed", processed, "total", total, "percent", progress)
		}
		return nil
	})
	if err != nil {
		observability.DefaultMetrics.ReplayFailures.Add(1)
		slog.Error("Replay failed", "component", "replay", "scope", scope, "processed", processed, "error", err)
		return err
	}

	if opts.AggregateID == "" && r.projectionManager != nil {
		if err := r.projectionManager.MarkReplayComplete(); err != nil {
			observability.DefaultMetrics.ReplayFailures.Add(1)
			slog.Error("Replay completed but projection version update failed",
				"component", "replay",
				"scope", scope,
				"events", processed,
				"error", err)
			return err
		}
	}

	slog.Info("Replay completed", "component", "replay", "scope", scope, "events", processed, "duration", time.Since(start))
	return nil
}

// dispatch routes a hydrated event to the same projection method the Kafka
// consumer uses. Keeping the type switch here — rather than introducing a
// new abstraction — makes it obvious that replay and live consumption share
// one and only one projection path.
func (r *Replayer) dispatch(ev ReplayEvent) error {
	switch e := ev.Event.(type) {
	case *commonevents.WalletCreatedEvent:
		return r.eventHandler.OnWalletCreated(e)
	case *commonevents.WalletCreditedEvent:
		return r.eventHandler.OnWalletCredited(e)
	case *commonevents.WalletDebitedEvent:
		return r.eventHandler.OnWalletDebited(e)
	default:
		return fmt.Errorf("no replay projection for event type %q", ev.EventType)
	}
}

// resetReadModel clears the tables replay is about to rebuild so that
// projections start from a clean slate. The processed_events inbox is
// also cleared in scope so replay is not short-circuited as "already
// processed" by applyIdempotent.
//
// Operational tables such as dead_letter_events are deliberately left
// untouched: replay rebuilds the read model, not the failure history.
//
// Scoping:
//   - Full replay: truncate wallets, transactions, processed_events.
//   - Aggregate-scoped replay: delete only rows belonging to that aggregate.
func (r *Replayer) resetReadModel(aggregateID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if aggregateID == "" {
			// TRUNCATE is cheaper than DELETE for a full wipe.
			// RESTART IDENTITY / CASCADE are not needed here — no sequences
			// or FKs are involved.
			if err := tx.Exec(
				`TRUNCATE TABLE ledger_entries, transactions, wallets, processed_events`,
			).Error; err != nil {
				return err
			}
			return nil
		}

		if err := tx.Where("wallet_id = ?", aggregateID).
			Delete(&domain.LedgerEntry{}).Error; err != nil {
			return err
		}
		if err := tx.Where("wallet_id = ?", aggregateID).
			Delete(&domain.Transaction{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", aggregateID).
			Delete(&domain.Wallet{}).Error; err != nil {
			return err
		}
		if err := tx.Where("aggregate_id = ?", aggregateID).
			Delete(&ProcessedEvent{}).Error; err != nil {
			return err
		}
		return nil
	})
}
