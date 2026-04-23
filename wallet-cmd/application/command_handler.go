package application

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	corehandlers "github.com/tunadonmez/go-cqrs-es/cqrs-core/handlers"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/domain"
)

// CommandHandler processes all wallet commands.
type CommandHandler struct {
	eventSourcingHandler corehandlers.EventSourcingHandler[domain.WalletAggregate]
}

func NewCommandHandler(esh corehandlers.EventSourcingHandler[domain.WalletAggregate]) *CommandHandler {
	return &CommandHandler{eventSourcingHandler: esh}
}

func (h *CommandHandler) HandleCreateWallet(cmd *commands.CreateWalletCommand) error {
	aggregate, err := domain.NewWalletAggregateFromCommand(cmd)
	if err != nil {
		return err
	}
	return h.eventSourcingHandler.Save(aggregate)
}

func (h *CommandHandler) HandleCreditWallet(cmd *commands.CreditWalletCommand) error {
	aggregate, err := h.eventSourcingHandler.GetByID(cmd.ID)
	if err != nil {
		return err
	}
	if err := aggregate.Credit(cmd.Amount, cmd.Reference, cmd.Description); err != nil {
		return err
	}
	return h.eventSourcingHandler.Save(aggregate)
}

func (h *CommandHandler) HandleDebitWallet(cmd *commands.DebitWalletCommand) error {
	aggregate, err := h.eventSourcingHandler.GetByID(cmd.ID)
	if err != nil {
		return err
	}
	if err := aggregate.Debit(cmd.Amount, cmd.Reference, cmd.Description); err != nil {
		return err
	}
	return h.eventSourcingHandler.Save(aggregate)
}

func (h *CommandHandler) HandleTransferFunds(cmd *commands.TransferFundsCommand) error {
	sourceWallet, err := h.eventSourcingHandler.GetByID(cmd.ID)
	if err != nil {
		return err
	}
	destinationWallet, err := h.eventSourcingHandler.GetByID(cmd.DestinationWalletID)
	if err != nil {
		return fmt.Errorf("failed to load destination wallet: %w", err)
	}

	reference := strings.TrimSpace(cmd.Reference)
	if reference == "" {
		reference = uuid.New().String()
	}

	if err := sourceWallet.TransferTo(destinationWallet, cmd.Amount, reference, cmd.Description); err != nil {
		return err
	}
	if err := h.eventSourcingHandler.Save(sourceWallet); err != nil {
		return err
	}
	return h.eventSourcingHandler.Save(destinationWallet)
}
