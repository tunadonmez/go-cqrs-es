package infrastructure

import (
	"log/slog"

	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
	corehandlers "github.com/tunadonmez/go-cqrs-es/cqrs-core/handlers"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/domain"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/observability"
)

// WalletEventSourcingHandler rehydrates WalletAggregates from the event store.
type WalletEventSourcingHandler struct {
	eventStore coreinfra.EventStore
}

type snapshotCapableEventStore interface {
	GetSnapshot(aggregateID string) (*domain.WalletAggregateSnapshot, error)
	GetEventsAfterVersion(aggregateID string, afterVersion int) ([]corevents.BaseEvent, error)
	SaveSnapshotIfDue(aggregate *domain.WalletAggregate) error
}

func NewWalletEventSourcingHandler(es coreinfra.EventStore) *WalletEventSourcingHandler {
	return &WalletEventSourcingHandler{eventStore: es}
}

func (h *WalletEventSourcingHandler) Save(aggregate *domain.WalletAggregate) error {
	changes := aggregate.GetUncommittedChanges()
	err := h.eventStore.SaveEvents(aggregate.ID, changes, aggregate.Version)
	if err != nil {
		return err
	}
	if len(changes) > 0 {
		aggregate.Version = changes[len(changes)-1].GetVersion()
	}
	if snapshotStore, ok := h.eventStore.(snapshotCapableEventStore); ok && len(changes) > 0 {
		if err := snapshotStore.SaveSnapshotIfDue(aggregate); err != nil {
			slog.Error("Aggregate snapshot creation failed",
				"aggregateId", aggregate.ID,
				"aggregateType", "domain.WalletAggregate",
				"version", aggregate.Version,
				"error", err)
		}
	}
	aggregate.MarkChangesAsCommitted()
	return nil
}

func (h *WalletEventSourcingHandler) GetByID(id string) (*domain.WalletAggregate, error) {
	aggregate := domain.NewWalletAggregate()

	if snapshotStore, ok := h.eventStore.(snapshotCapableEventStore); ok {
		snapshot, err := snapshotStore.GetSnapshot(id)
		if err != nil {
			slog.Warn("Aggregate snapshot unavailable; falling back to full replay",
				"aggregateId", id,
				"aggregateType", "domain.WalletAggregate",
				"error", err)
		} else if snapshot != nil {
			aggregate.Restore(*snapshot)
			observability.DefaultMetrics.SnapshotsLoaded.Add(1)
			slog.Info("Aggregate snapshot loaded",
				"aggregateId", id,
				"aggregateType", "domain.WalletAggregate",
				"version", snapshot.Version)

			events, err := snapshotStore.GetEventsAfterVersion(id, snapshot.Version)
			if err != nil {
				return nil, err
			}
			replayAggregate(aggregate, events, snapshot.Version)
			return aggregate, nil
		}
	}

	observability.DefaultMetrics.SnapshotFullReplays.Add(1)
	slog.Info("Aggregate snapshot not found; falling back to full replay",
		"aggregateId", id,
		"aggregateType", "domain.WalletAggregate")

	events, err := h.eventStore.GetEvents(id)
	if err != nil {
		return nil, err
	}
	replayAggregate(aggregate, events, -1)
	return aggregate, nil
}

func replayAggregate(aggregate *domain.WalletAggregate, events []corevents.BaseEvent, baseVersion int) {
	if len(events) == 0 {
		aggregate.Version = baseVersion
		return
	}
	aggregate.ReplayEvents(aggregate, events)
	latestVersion := baseVersion
	for _, e := range events {
		if v := e.GetVersion(); v > latestVersion {
			latestVersion = v
		}
	}
	aggregate.Version = latestVersion
}

var _ corehandlers.EventSourcingHandler[domain.WalletAggregate] = (*WalletEventSourcingHandler)(nil)
