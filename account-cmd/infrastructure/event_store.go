package infrastructure

import (
	"errors"
	"fmt"
	"time"

	"github.com/tunadonmez/go-cqrs-es/account-cmd/domain"
	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
	"github.com/tunadonmez/go-cqrs-es/cqrs-core/producers"
)

const EventsTopic = "BankAccountEvents"

var (
	ErrAggregateNotFound = errors.New("incorrect account ID provided")
	ErrConcurrency       = errors.New("a newer version of this aggregate exists in the event store")
)

// AccountEventStore is the write-side event store backed by MongoDB.
type AccountEventStore struct {
	producer   producers.EventProducer
	repository *EventStoreRepository
}

func NewAccountEventStore(producer producers.EventProducer, repo *EventStoreRepository) *AccountEventStore {
	return &AccountEventStore{producer: producer, repository: repo}
}

func (s *AccountEventStore) SaveEvents(aggregateID string, evts []corevents.BaseEvent, expectedVersion int) error {
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
			AggregateType:       fmt.Sprintf("%T", domain.AccountAggregate{}),
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

func (s *AccountEventStore) GetEvents(aggregateID string) ([]corevents.BaseEvent, error) {
	eventStream, err := s.repository.FindByAggregateIdentifier(aggregateID)
	if err != nil {
		return nil, err
	}
	if len(eventStream) == 0 {
		return nil, ErrAggregateNotFound
	}
	result := make([]corevents.BaseEvent, len(eventStream))
	for i, doc := range eventStream {
		result[i] = doc.EventData
	}
	return result, nil
}
