package infrastructure

import (
	"errors"
	"fmt"
	"time"

	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
	"github.com/tunadonmez/go-cqrs-es/cqrs-core/producers"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/domain"
)

const EventsTopic = "WalletEvents"

var (
	ErrWalletNotFound = errors.New("incorrect wallet ID provided")
	ErrConcurrency    = errors.New("a newer version of this aggregate exists in the event store")
)

// WalletEventStore is the write-side event store backed by MongoDB.
type WalletEventStore struct {
	producer   producers.EventProducer
	repository *EventStoreRepository
}

func NewWalletEventStore(producer producers.EventProducer, repo *EventStoreRepository) *WalletEventStore {
	return &WalletEventStore{producer: producer, repository: repo}
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

		model := &EventModelDoc{
			TimeStamp:           time.Now(),
			AggregateIdentifier: aggregateID,
			AggregateType:       fmt.Sprintf("%T", domain.WalletAggregate{}),
			Version:             version,
			EventType:           event.EventTypeName(),
			EventData:           event,
		}

		persisted, err := s.repository.Save(model)
		if err != nil {
			return err
		}
		if !persisted.ID.IsZero() {
			if err := s.producer.Produce(EventsTopic, event); err != nil {
				return err
			}
		}
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
