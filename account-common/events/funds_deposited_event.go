package events

import corevents "github.com/techbank/cqrs-core/events"

func init() {
	corevents.Register("FundsDepositedEvent", func() corevents.BaseEvent { return &FundsDepositedEvent{} })
}

type FundsDepositedEvent struct {
	corevents.BaseEventData
	Amount float64 `json:"amount" bson:"amount"`
}

func (e *FundsDepositedEvent) EventTypeName() string { return "FundsDepositedEvent" }
