package infrastructure

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/domain"
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
			EventID:             event.GetEventID(),
			EventType:           event.EventTypeName(),
			EventData:           event,
			PublishStatus:       PublishStatusPending,
		}

		if _, err := s.repository.Save(model); err != nil {
			slog.Error("Failed to save event to store",
				"aggregateId", aggregateID,
				"eventId", event.GetEventID(),
				"type", event.EventTypeName(),
				"error", err)
			return err
		}
		slog.Info("Event persisted to event store",
			"aggregateId", aggregateID,
			"eventId", event.GetEventID(),
			"type", event.EventTypeName(),
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
