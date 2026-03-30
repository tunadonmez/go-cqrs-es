package events

import corevents "github.com/techbank/cqrs-core/events"

func init() {
	corevents.Register("FundsWithdrawnEvent", func() corevents.BaseEvent { return &FundsWithdrawnEvent{} })
}

type FundsWithdrawnEvent struct {
	corevents.BaseEventData
	Amount float64 `json:"amount" bson:"amount"`
}

func (e *FundsWithdrawnEvent) EventTypeName() string { return "FundsWithdrawnEvent" }
