package events

import corevents "github.com/techbank/cqrs-core/events"

func init() {
	corevents.Register("AccountClosedEvent", func() corevents.BaseEvent { return &AccountClosedEvent{} })
}

type AccountClosedEvent struct {
	corevents.BaseEventData
}

func (e *AccountClosedEvent) EventTypeName() string { return "AccountClosedEvent" }
