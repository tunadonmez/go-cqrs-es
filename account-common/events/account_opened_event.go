package events

import (
	"time"

	"github.com/techbank/account-common/dto"
	corevents "github.com/techbank/cqrs-core/events"
)

func init() {
	corevents.Register("AccountOpenedEvent", func() corevents.BaseEvent { return &AccountOpenedEvent{} })
}

type AccountOpenedEvent struct {
	corevents.BaseEventData
	AccountHolder  string          `json:"accountHolder" bson:"accountHolder"`
	AccountType    dto.AccountType `json:"accountType" bson:"accountType"`
	CreatedDate    time.Time       `json:"createdDate" bson:"createdDate"`
	OpeningBalance float64         `json:"openingBalance" bson:"openingBalance"`
}

func (e *AccountOpenedEvent) EventTypeName() string { return "AccountOpenedEvent" }
