package infrastructure

import "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"

// EventStore defines persistence for domain events.
type EventStore interface {
	SaveEvents(aggregateID string, events []events.BaseEvent, expectedVersion int) error
	GetEvents(aggregateID string) ([]events.BaseEvent, error)
}
