package infrastructure

import "github.com/techbank/cqrs-core/events"

// EventStore defines persistence for domain events.
type EventStore interface {
	SaveEvents(aggregateID string, evts []events.BaseEvent, expectedVersion int) error
	GetEvents(aggregateID string) ([]events.BaseEvent, error)
}
