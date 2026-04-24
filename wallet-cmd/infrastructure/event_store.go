package infrastructure

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/domain"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/observability"
)

// EventsTopic is the Kafka topic all wallet events are published to.
const EventsTopic = "WalletEvents"

var (
	ErrWalletNotFound = errors.New("incorrect wallet ID provided")
	ErrConcurrency    = errors.New("a newer version of this aggregate exists in the event store")
)

// WalletEventStore is the write-side event store backed by MongoDB.
//
// Each SaveEvents call persists the events together with a PENDING outbox
// entry in the same document; Kafka publishing has been moved out of the
// command path and is handled asynchronously by OutboxPublisher.
type WalletEventStore struct {
	repository *EventStoreRepository
}

const snapshotThreshold = 50

func NewWalletEventStore(repo *EventStoreRepository) *WalletEventStore {
	return &WalletEventStore{repository: repo}
}

func (s *WalletEventStore) SaveEvents(aggregateID string, evts []corevents.BaseEvent, expectedVersion int) error {
	eventStream, err := s.repository.FindByAggregateIdentifier(aggregateID)
	if err != nil {
		return err
	}

	if expectedVersion != -1 {
		if len(eventStream) == 0 || eventStream[len(eventStream)-1].Version != expectedVersion {
			return ErrConcurrency
		}
	}

	version := expectedVersion
	for _, event := range evts {
		version++
		event.SetVersion(version)
		corevents.EnsureSchemaVersion(event)
		if event.GetAggregateID() == "" {
			event.SetAggregateID(aggregateID)
		}
		if event.GetEventID() == "" {
			return fmt.Errorf("event %s is missing an event id", event.EventTypeName())
		}

		model := &EventModelDoc{
			TimeStamp:           time.Now().UTC(),
			AggregateIdentifier: aggregateID,
			AggregateType:       fmt.Sprintf("%T", domain.WalletAggregate{}),
			Version:             version,
			EventSchemaVersion:  event.GetSchemaVersion(),
			EventID:             event.GetEventID(),
			EventType:           event.EventTypeName(),
			EventData:           event,
			PublishStatus:       PublishStatusPending,
		}

		if _, err := s.repository.Save(model); err != nil {
			observability.DefaultMetrics.EventPersistFailures.Add(1)
			slog.Error("Event persistence failed",
				"aggregateId", aggregateID,
				"eventId", event.GetEventID(),
				"aggregateType", model.AggregateType,
				"eventType", event.EventTypeName(),
				"version", version,
				"error", err)
			return err
		}
		observability.DefaultMetrics.EventsPersisted.Add(1)
		slog.Info("Event persisted",
			"aggregateId", aggregateID,
			"eventId", event.GetEventID(),
			"aggregateType", model.AggregateType,
			"eventType", event.EventTypeName(),
			"version", version)
	}
	return nil
}

func (s *WalletEventStore) GetEvents(aggregateID string) ([]corevents.BaseEvent, error) {
	eventStream, err := s.repository.FindByAggregateIdentifier(aggregateID)
	if err != nil {
		return nil, err
	}
	if len(eventStream) == 0 {
		return nil, ErrWalletNotFound
	}
	result := make([]corevents.BaseEvent, len(eventStream))
	for i, doc := range eventStream {
		result[i] = doc.EventData
	}
	return result, nil
}

func (s *WalletEventStore) GetEventsAfterVersion(aggregateID string, afterVersion int) ([]corevents.BaseEvent, error) {
	eventStream, err := s.repository.FindByAggregateIdentifierAfterVersion(aggregateID, afterVersion)
	if err != nil {
		return nil, err
	}
	result := make([]corevents.BaseEvent, len(eventStream))
	for i, doc := range eventStream {
		result[i] = doc.EventData
	}
	return result, nil
}

func (s *WalletEventStore) GetSnapshot(aggregateID string) (*domain.WalletAggregateSnapshot, error) {
	snapshot := &domain.WalletAggregateSnapshot{}
	found, _, err := s.repository.LoadLatestSnapshot(aggregateID, snapshot)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return snapshot, nil
}

func (s *WalletEventStore) SaveSnapshotIfDue(aggregate *domain.WalletAggregate) error {
	if aggregate == nil || aggregate.ID == "" || aggregate.Version < 0 {
		return nil
	}
	if (aggregate.Version+1)%snapshotThreshold != 0 {
		return nil
	}

	snapshot := aggregate.Snapshot()
	if err := s.repository.SaveSnapshot(
		aggregate.ID,
		fmt.Sprintf("%T", domain.WalletAggregate{}),
		aggregate.Version,
		snapshot,
	); err != nil {
		return err
	}
	observability.DefaultMetrics.SnapshotsCreated.Add(1)
	slog.Info("Aggregate snapshot created",
		"aggregateId", aggregate.ID,
		"aggregateType", fmt.Sprintf("%T", domain.WalletAggregate{}),
		"version", aggregate.Version,
		"threshold", snapshotThreshold)
	return nil
}
