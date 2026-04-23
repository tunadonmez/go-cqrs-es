package events

import (
	"time"

	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
)

func init() {
	corevents.Register("WalletDebitedEvent", func() corevents.BaseEvent { return &WalletDebitedEvent{} })
}

type WalletDebitedEvent struct {
	corevents.BaseEventData
	Amount               float64             `json:"amount" bson:"amount"`
	TransactionType      dto.TransactionType `json:"transactionType" bson:"transactionType"`
	CounterpartyWalletID string              `json:"counterpartyWalletId,omitempty" bson:"counterpartyWalletId,omitempty"`
	Reference            string              `json:"reference,omitempty" bson:"reference,omitempty"`
	Description          string              `json:"description,omitempty" bson:"description,omitempty"`
	OccurredAt           time.Time           `json:"occurredAt" bson:"occurredAt"`
}

func (e *WalletDebitedEvent) EventTypeName() string { return "WalletDebitedEvent" }
