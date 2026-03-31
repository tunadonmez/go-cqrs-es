package producers

import "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"

// EventProducer publishes domain events to a message broker.
type EventProducer interface {
	Produce(topic string, event events.BaseEvent) error
}
