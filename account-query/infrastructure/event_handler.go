package infrastructure

import (
	commonevents "github.com/techbank/account-common/events"
	"github.com/techbank/account-query/domain"
)

// AccountEventHandler projects domain events onto the MySQL read model.
type AccountEventHandler struct {
	repository *AccountRepository
}

func NewAccountEventHandler(repo *AccountRepository) *AccountEventHandler {
	return &AccountEventHandler{repository: repo}
}

func (h *AccountEventHandler) OnAccountOpened(event *commonevents.AccountOpenedEvent) error {
	account := &domain.Account{
		ID:            event.GetID(),
		AccountHolder: event.AccountHolder,
		CreationDate:  event.CreatedDate,
		AccountType:   event.AccountType,
		Balance:       event.OpeningBalance,
	}
	return h.repository.Save(account)
}

func (h *AccountEventHandler) OnFundsDeposited(event *commonevents.FundsDepositedEvent) error {
	account, err := h.repository.FindByID(event.GetID())
	if err != nil {
		return nil // account not found; skip
	}
	account.Balance += event.Amount
	return h.repository.Save(account)
}

func (h *AccountEventHandler) OnFundsWithdrawn(event *commonevents.FundsWithdrawnEvent) error {
	account, err := h.repository.FindByID(event.GetID())
	if err != nil {
		return nil
	}
	account.Balance -= event.Amount
	return h.repository.Save(account)
}

func (h *AccountEventHandler) OnAccountClosed(event *commonevents.AccountClosedEvent) error {
	return h.repository.DeleteByID(event.GetID())
}
