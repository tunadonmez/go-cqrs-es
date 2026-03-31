package domain

import (
	"time"

	"github.com/tunadonmez/go-cqrs-es/account-common/dto"
)

// Account is the MySQL read-model entity.
type Account struct {
	ID            string          `json:"id" gorm:"primaryKey"`
	AccountHolder string          `json:"accountHolder"`
	CreationDate  time.Time       `json:"creationDate"`
	AccountType   dto.AccountType `json:"accountType"`
	Balance       float64         `json:"balance"`
}

func (a *Account) EntityID() string { return a.ID }
