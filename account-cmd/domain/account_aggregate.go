package domain

import (
	"errors"
	"time"

	"github.com/tunadonmez/go-cqrs-es/account-cmd/api/commands"
	commonevents "github.com/tunadonmez/go-cqrs-es/account-common/events"
	coredomain "github.com/tunadonmez/go-cqrs-es/cqrs-core/domain"
)

// AccountAggregate is the write-side aggregate for bank accounts.
type AccountAggregate struct {
	coredomain.AggregateRoot
	active  bool
	balance float64
}

func NewAccountAggregate() *AccountAggregate {
	return &AccountAggregate{
		AggregateRoot: *coredomain.NewAggregateRoot(),
	}
}

func NewAccountAggregateFromCommand(cmd *commands.OpenAccountCommand) (*AccountAggregate, error) {
	if cmd.OpeningBalance < 0 {
		return nil, errors.New("the opening balance cannot be negative")
	}
	a := NewAccountAggregate()
	event := &commonevents.AccountOpenedEvent{
		AccountHolder:  cmd.AccountHolder,
		AccountType:    cmd.AccountType,
		CreatedDate:    time.Now(),
		OpeningBalance: cmd.OpeningBalance,
	}
	event.SetID(cmd.ID)
	a.RaiseEvent(a, event)
	return a, nil
}

func (a *AccountAggregate) ApplyAccountOpenedEvent(event *commonevents.AccountOpenedEvent) {
	a.ID = event.GetID()
	a.active = true
	a.balance = event.OpeningBalance
}

func (a *AccountAggregate) DepositFunds(amount float64) error {
	if !a.active {
		return errors.New("funds cannot be deposited into a closed account")
	}
	if amount <= 0 {
		return errors.New("the deposit amount must be greater than 0")
	}
	event := &commonevents.FundsDepositedEvent{Amount: amount}
	event.SetID(a.ID)
	a.RaiseEvent(a, event)
	return nil
}

func (a *AccountAggregate) ApplyFundsDepositedEvent(event *commonevents.FundsDepositedEvent) {
	a.ID = event.GetID()
	a.balance += event.Amount
}

func (a *AccountAggregate) WithdrawFunds(amount float64) error {
	if !a.active {
		return errors.New("funds cannot be withdrawn from a closed account")
	}
	if amount <= 0 {
		return errors.New("the withdrawal amount must be greater than 0")
	}
	if amount > a.balance {
		return errors.New("withdrawal declined, insufficient funds")
	}
	event := &commonevents.FundsWithdrawnEvent{Amount: amount}
	event.SetID(a.ID)
	a.RaiseEvent(a, event)
	return nil
}

func (a *AccountAggregate) ApplyFundsWithdrawnEvent(event *commonevents.FundsWithdrawnEvent) {
	a.ID = event.GetID()
	a.balance -= event.Amount
}

func (a *AccountAggregate) CloseAccount() error {
	if !a.active {
		return errors.New("the bank account has already been closed")
	}
	event := &commonevents.AccountClosedEvent{}
	event.SetID(a.ID)
	a.RaiseEvent(a, event)
	return nil
}

func (a *AccountAggregate) ApplyAccountClosedEvent(event *commonevents.AccountClosedEvent) {
	a.ID = event.GetID()
	a.active = false
}
