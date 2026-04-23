package application

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	corehandlers "github.com/tunadonmez/go-cqrs-es/cqrs-core/handlers"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/domain"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/observability"
)

// CommandHandler processes all wallet commands.
type CommandHandler struct {
	eventSourcingHandler corehandlers.EventSourcingHandler[domain.WalletAggregate]
}

func NewCommandHandler(esh corehandlers.EventSourcingHandler[domain.WalletAggregate]) *CommandHandler {
	return &CommandHandler{eventSourcingHandler: esh}
}

func (h *CommandHandler) HandleCreateWallet(cmd *commands.CreateWalletCommand) error {
	slog.Info("Command handling started",
		"commandType", "CreateWalletCommand",
		"commandId", cmd.GetID(),
		"aggregateId", cmd.GetID())
	aggregate, err := domain.NewWalletAggregateFromCommand(cmd)
	if err != nil {
		return logCommandFailure("CreateWalletCommand", cmd.GetID(), cmd.GetID(), err)
	}
	if err := h.eventSourcingHandler.Save(aggregate); err != nil {
		return logCommandFailure("CreateWalletCommand", cmd.GetID(), cmd.GetID(), err)
	}
	logCommandSuccess("CreateWalletCommand", cmd.GetID(), cmd.GetID())
	return nil
}

func (h *CommandHandler) HandleCreditWallet(cmd *commands.CreditWalletCommand) error {
	slog.Info("Command handling started",
		"commandType", "CreditWalletCommand",
		"commandId", cmd.GetID(),
		"aggregateId", cmd.ID)
	aggregate, err := h.eventSourcingHandler.GetByID(cmd.ID)
	if err != nil {
		return logCommandFailure("CreditWalletCommand", cmd.GetID(), cmd.ID, err)
	}
	if err := aggregate.Credit(cmd.Amount, cmd.Reference, cmd.Description); err != nil {
		return logCommandFailure("CreditWalletCommand", cmd.GetID(), cmd.ID, err)
	}
	if err := h.eventSourcingHandler.Save(aggregate); err != nil {
		return logCommandFailure("CreditWalletCommand", cmd.GetID(), cmd.ID, err)
	}
	logCommandSuccess("CreditWalletCommand", cmd.GetID(), cmd.ID)
	return nil
}

func (h *CommandHandler) HandleDebitWallet(cmd *commands.DebitWalletCommand) error {
	slog.Info("Command handling started",
		"commandType", "DebitWalletCommand",
		"commandId", cmd.GetID(),
		"aggregateId", cmd.ID)
	aggregate, err := h.eventSourcingHandler.GetByID(cmd.ID)
	if err != nil {
		return logCommandFailure("DebitWalletCommand", cmd.GetID(), cmd.ID, err)
	}
	if err := aggregate.Debit(cmd.Amount, cmd.Reference, cmd.Description); err != nil {
		return logCommandFailure("DebitWalletCommand", cmd.GetID(), cmd.ID, err)
	}
	if err := h.eventSourcingHandler.Save(aggregate); err != nil {
		return logCommandFailure("DebitWalletCommand", cmd.GetID(), cmd.ID, err)
	}
	logCommandSuccess("DebitWalletCommand", cmd.GetID(), cmd.ID)
	return nil
}

func (h *CommandHandler) HandleTransferFunds(cmd *commands.TransferFundsCommand) error {
	slog.Info("Command handling started",
		"commandType", "TransferFundsCommand",
		"commandId", cmd.GetID(),
		"aggregateId", cmd.ID,
		"destinationAggregateId", cmd.DestinationWalletID)
	sourceWallet, err := h.eventSourcingHandler.GetByID(cmd.ID)
	if err != nil {
		return logCommandFailure("TransferFundsCommand", cmd.GetID(), cmd.ID, err,
			"destinationAggregateId", cmd.DestinationWalletID)
	}
	destinationWallet, err := h.eventSourcingHandler.GetByID(cmd.DestinationWalletID)
	if err != nil {
		return logCommandFailure("TransferFundsCommand", cmd.GetID(), cmd.ID,
			fmt.Errorf("failed to load destination wallet: %w", err),
			"destinationAggregateId", cmd.DestinationWalletID)
	}

	reference := strings.TrimSpace(cmd.Reference)
	if reference == "" {
		reference = uuid.New().String()
	}

	if err := sourceWallet.TransferTo(destinationWallet, cmd.Amount, reference, cmd.Description); err != nil {
		return logCommandFailure("TransferFundsCommand", cmd.GetID(), cmd.ID, err,
			"destinationAggregateId", cmd.DestinationWalletID)
	}
	if err := h.eventSourcingHandler.Save(sourceWallet); err != nil {
		return logCommandFailure("TransferFundsCommand", cmd.GetID(), cmd.ID, err,
			"destinationAggregateId", cmd.DestinationWalletID)
	}
	if err := h.eventSourcingHandler.Save(destinationWallet); err != nil {
		return logCommandFailure("TransferFundsCommand", cmd.GetID(), cmd.ID, err,
			"destinationAggregateId", cmd.DestinationWalletID)
	}
	logCommandSuccess("TransferFundsCommand", cmd.GetID(), cmd.ID,
		"destinationAggregateId", cmd.DestinationWalletID)
	return nil
}

func logCommandSuccess(commandType, commandID, aggregateID string, extra ...any) {
	observability.DefaultMetrics.CommandsSucceeded.Add(1)
	attrs := []any{
		"commandType", commandType,
		"commandId", commandID,
		"aggregateId", aggregateID,
	}
	attrs = append(attrs, extra...)
	slog.Info("Command handled", attrs...)
}

func logCommandFailure(commandType, commandID, aggregateID string, err error, extra ...any) error {
	observability.DefaultMetrics.CommandFailures.Add(1)
	attrs := []any{
		"commandType", commandType,
		"commandId", commandID,
		"aggregateId", aggregateID,
		"error", err,
	}
	attrs = append(attrs, extra...)
	slog.Error("Command failed", attrs...)
	return err
}
