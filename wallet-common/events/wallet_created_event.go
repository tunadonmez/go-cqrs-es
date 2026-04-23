package events

import (
	"time"

	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
)

func init() {
	corevents.Register("WalletCreatedEvent", func() corevents.BaseEvent { return &WalletCreatedEvent{} })
}

type WalletCreatedEvent struct {
	corevents.BaseEventData
	Owner          string    `json:"owner" bson:"owner"`
	Currency       string    `json:"currency" bson:"currency"`
	CreatedAt      time.Time `json:"createdAt" bson:"createdAt"`
	OpeningBalance float64   `json:"openingBalance" bson:"openingBalance"`
}

func (e *WalletCreatedEvent) EventTypeName() string { return "WalletCreatedEvent" }
