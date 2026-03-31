package application

import (
	"github.com/tunadonmez/go-cqrs-es/account-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/account-cmd/domain"
	corehandlers "github.com/tunadonmez/go-cqrs-es/cqrs-core/handlers"
)

// CommandHandler processes all account commands.
type CommandHandler struct {
	eventSourcingHandler corehandlers.EventSourcingHandler[domain.AccountAggregate]
}

func NewCommandHandler(esh corehandlers.EventSourcingHandler[domain.AccountAggregate]) *CommandHandler {
	return &CommandHandler{eventSourcingHandler: esh}
}

func (h *CommandHandler) HandleOpenAccount(cmd *commands.OpenAccountCommand) error {
	aggregate, err := domain.NewAccountAggregateFromCommand(cmd)
	if err != nil {
		return err
	}
	return h.eventSourcingHandler.Save(aggregate)
}

func (h *CommandHandler) HandleDepositFunds(cmd *commands.DepositFundsCommand) error {
	aggregate, err := h.eventSourcingHandler.GetByID(cmd.ID)
	if err != nil {
		return err
	}
	if err := aggregate.DepositFunds(cmd.Amount); err != nil {
		return err
	}
	return h.eventSourcingHandler.Save(aggregate)
}

func (h *CommandHandler) HandleWithdrawFunds(cmd *commands.WithdrawFundsCommand) error {
	aggregate, err := h.eventSourcingHandler.GetByID(cmd.ID)
	if err != nil {
		return err
	}
	if err := aggregate.WithdrawFunds(cmd.Amount); err != nil {
		return err
	}
	return h.eventSourcingHandler.Save(aggregate)
}

func (h *CommandHandler) HandleCloseAccount(cmd *commands.CloseAccountCommand) error {
	aggregate, err := h.eventSourcingHandler.GetByID(cmd.ID)
	if err != nil {
		return err
	}
	if err := aggregate.CloseAccount(); err != nil {
		return err
	}
	return h.eventSourcingHandler.Save(aggregate)
}
