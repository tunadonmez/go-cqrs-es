package domain

import (
	"errors"
	"strings"
	"time"

	coredomain "github.com/tunadonmez/go-cqrs-es/cqrs-core/domain"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
	commonevents "github.com/tunadonmez/go-cqrs-es/wallet-common/events"
)

// WalletAggregate is the write-side aggregate for wallets.
type WalletAggregate struct {
	coredomain.AggregateRoot
	owner     string
	currency  string
	createdAt time.Time
	balance   float64
}

func NewWalletAggregate() *WalletAggregate {
	return &WalletAggregate{
		AggregateRoot: *coredomain.NewAggregateRoot(),
	}
}

func NewWalletAggregateFromCommand(cmd *commands.CreateWalletCommand) (*WalletAggregate, error) {
	owner := strings.TrimSpace(cmd.Owner)
	currency := strings.ToUpper(strings.TrimSpace(cmd.Currency))
	if owner == "" {
		return nil, errors.New("wallet owner is required")
	}
	if currency == "" {
		return nil, errors.New("wallet currency is required")
	}
	if cmd.OpeningBalance < 0 {
		return nil, errors.New("the opening balance cannot be negative")
	}
	w := NewWalletAggregate()
	event := &commonevents.WalletCreatedEvent{
		Owner:          owner,
		Currency:       currency,
		CreatedAt:      time.Now().UTC(),
		OpeningBalance: cmd.OpeningBalance,
	}
	event.SetAggregateID(cmd.ID)
	w.RaiseEvent(w, event)
	return w, nil
}

func (w *WalletAggregate) ApplyWalletCreatedEvent(event *commonevents.WalletCreatedEvent) {
	w.ID = event.GetAggregateID()
	w.owner = event.Owner
	w.currency = event.Currency
	w.createdAt = event.CreatedAt
	w.balance = event.OpeningBalance
}

func (w *WalletAggregate) Credit(amount float64, reference, description string) error {
	return w.credit(amount, dto.TransactionTypeCredit, "", reference, description, time.Now().UTC())
}

func (w *WalletAggregate) Debit(amount float64, reference, description string) error {
	return w.debit(amount, dto.TransactionTypeDebit, "", reference, description, time.Now().UTC())
}

func (w *WalletAggregate) TransferTo(destination *WalletAggregate, amount float64, reference, description string) error {
	if destination == nil || destination.ID == "" {
		return errors.New("destination wallet is required")
	}
	if w.ID == destination.ID {
		return errors.New("source and destination wallets must be different")
	}
	if w.currency != destination.currency {
		return errors.New("transfer currency mismatch between wallets")
	}
	occurredAt := time.Now().UTC()
	if err := w.debit(amount, dto.TransactionTypeTransferOut, destination.ID, reference, description, occurredAt); err != nil {
		return err
	}
	return destination.credit(amount, dto.TransactionTypeTransferIn, w.ID, reference, description, occurredAt)
}

func (w *WalletAggregate) credit(amount float64, transactionType dto.TransactionType, counterpartyWalletID, reference, description string, occurredAt time.Time) error {
	if amount <= 0 {
		return errors.New("the credit amount must be greater than 0")
	}
	event := &commonevents.WalletCreditedEvent{
		Amount:               amount,
		TransactionType:      transactionType,
		CounterpartyWalletID: strings.TrimSpace(counterpartyWalletID),
		Reference:            strings.TrimSpace(reference),
		Description:          strings.TrimSpace(description),
		OccurredAt:           occurredAt,
	}
	event.SetAggregateID(w.ID)
	w.RaiseEvent(w, event)
	return nil
}

func (w *WalletAggregate) ApplyWalletCreditedEvent(event *commonevents.WalletCreditedEvent) {
	w.ID = event.GetAggregateID()
	w.balance += event.Amount
}

func (w *WalletAggregate) debit(amount float64, transactionType dto.TransactionType, counterpartyWalletID, reference, description string, occurredAt time.Time) error {
	if amount <= 0 {
		return errors.New("the debit amount must be greater than 0")
	}
	if amount > w.balance {
		return errors.New("debit declined, insufficient funds")
	}
	event := &commonevents.WalletDebitedEvent{
		Amount:               amount,
		TransactionType:      transactionType,
		CounterpartyWalletID: strings.TrimSpace(counterpartyWalletID),
		Reference:            strings.TrimSpace(reference),
		Description:          strings.TrimSpace(description),
		OccurredAt:           occurredAt,
	}
	event.SetAggregateID(w.ID)
	w.RaiseEvent(w, event)
	return nil
}

func (w *WalletAggregate) ApplyWalletDebitedEvent(event *commonevents.WalletDebitedEvent) {
	w.ID = event.GetAggregateID()
	w.balance -= event.Amount
}
