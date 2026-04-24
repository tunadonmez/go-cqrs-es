package infrastructure

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
	commonevents "github.com/tunadonmez/go-cqrs-es/wallet-common/events"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/observability"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WalletEventHandler projects domain events onto PostgreSQL read models.
//
// All projection calls run through applyIdempotent, which guarantees
// the projection side-effects and the "event processed" record commit
// atomically in a single transaction. Duplicate deliveries short-circuit
// before any read-model mutation happens.
type WalletEventHandler struct {
	repository *WalletRepository
}

func NewWalletEventHandler(repo *WalletRepository) *WalletEventHandler {
	return &WalletEventHandler{repository: repo}
}

func (h *WalletEventHandler) OnWalletCreated(event *commonevents.WalletCreatedEvent) error {
	return h.applyIdempotent(event, func(tx *gorm.DB) error {
		wallet := &domain.Wallet{
			ID:        event.GetAggregateID(),
			Owner:     event.Owner,
			Currency:  event.Currency,
			CreatedAt: event.CreatedAt,
			Balance:   event.OpeningBalance,
		}

		transaction := &domain.Transaction{
			ID:           transactionID(event.GetAggregateID(), event.GetVersion()),
			WalletID:     event.GetAggregateID(),
			Type:         dto.TransactionTypeOpeningBalance,
			Amount:       event.OpeningBalance,
			Description:  "wallet created",
			BalanceAfter: event.OpeningBalance,
			OccurredAt:   event.CreatedAt,
			EventVersion: event.GetVersion(),
		}

		if err := tx.Save(wallet).Error; err != nil {
			return err
		}
		if err := tx.Save(transaction).Error; err != nil {
			return err
		}

		entry := &domain.LedgerEntry{
			ID:              ledgerEntryID(event.GetEventID(), domain.LedgerEntryTypeCredit),
			WalletID:        event.GetAggregateID(),
			AggregateID:     event.GetAggregateID(),
			TransactionID:   transaction.ID,
			EventID:         event.GetEventID(),
			EventType:       event.EventTypeName(),
			EventVersion:    event.GetVersion(),
			TransactionType: dto.TransactionTypeOpeningBalance,
			EntryType:       domain.LedgerEntryTypeCredit,
			Amount:          event.OpeningBalance,
			Currency:        event.Currency,
			Reference:       "opening-balance",
			Description:     "wallet created",
			OccurredAt:      event.CreatedAt,
		}
		if err := h.repository.SaveLedgerEntriesTx(tx, entry); err != nil {
			return err
		}
		logLedgerProjected(event, 1)
		return nil
	})
}

func (h *WalletEventHandler) OnWalletCredited(event *commonevents.WalletCreditedEvent) error {
	return h.applyIdempotent(event, func(tx *gorm.DB) error {
		wallet, err := findWalletInTx(tx, event.GetAggregateID())
		if err != nil {
			return err
		}
		wallet.Balance += event.Amount

		transaction := &domain.Transaction{
			ID:                   transactionID(event.GetAggregateID(), event.GetVersion()),
			WalletID:             event.GetAggregateID(),
			Type:                 event.TransactionType,
			Amount:               event.Amount,
			CounterpartyWalletID: event.CounterpartyWalletID,
			Reference:            event.Reference,
			Description:          event.Description,
			BalanceAfter:         wallet.Balance,
			OccurredAt:           event.OccurredAt,
			EventVersion:         event.GetVersion(),
		}

		if err := tx.Save(wallet).Error; err != nil {
			return err
		}
		if err := tx.Save(transaction).Error; err != nil {
			return err
		}

		entry := &domain.LedgerEntry{
			ID:                   ledgerEntryID(event.GetEventID(), domain.LedgerEntryTypeCredit),
			WalletID:             event.GetAggregateID(),
			AggregateID:          event.GetAggregateID(),
			TransactionID:        transaction.ID,
			EventID:              event.GetEventID(),
			EventType:            event.EventTypeName(),
			EventVersion:         event.GetVersion(),
			TransactionType:      event.TransactionType,
			EntryType:            domain.LedgerEntryTypeCredit,
			Amount:               event.Amount,
			Currency:             wallet.Currency,
			CounterpartyWalletID: event.CounterpartyWalletID,
			Reference:            event.Reference,
			Description:          event.Description,
			OccurredAt:           event.OccurredAt,
		}
		if err := h.repository.SaveLedgerEntriesTx(tx, entry); err != nil {
			return err
		}
		if err := validateTransferLedgerInvariant(tx, entry); err != nil {
			return err
		}
		logLedgerProjected(event, 1)
		return nil
	})
}

func (h *WalletEventHandler) OnWalletDebited(event *commonevents.WalletDebitedEvent) error {
	return h.applyIdempotent(event, func(tx *gorm.DB) error {
		wallet, err := findWalletInTx(tx, event.GetAggregateID())
		if err != nil {
			return err
		}
		wallet.Balance -= event.Amount

		transaction := &domain.Transaction{
			ID:                   transactionID(event.GetAggregateID(), event.GetVersion()),
			WalletID:             event.GetAggregateID(),
			Type:                 event.TransactionType,
			Amount:               event.Amount,
			CounterpartyWalletID: event.CounterpartyWalletID,
			Reference:            event.Reference,
			Description:          event.Description,
			BalanceAfter:         wallet.Balance,
			OccurredAt:           event.OccurredAt,
			EventVersion:         event.GetVersion(),
		}

		if err := tx.Save(wallet).Error; err != nil {
			return err
		}
		if err := tx.Save(transaction).Error; err != nil {
			return err
		}

		entry := &domain.LedgerEntry{
			ID:                   ledgerEntryID(event.GetEventID(), domain.LedgerEntryTypeDebit),
			WalletID:             event.GetAggregateID(),
			AggregateID:          event.GetAggregateID(),
			TransactionID:        transaction.ID,
			EventID:              event.GetEventID(),
			EventType:            event.EventTypeName(),
			EventVersion:         event.GetVersion(),
			TransactionType:      event.TransactionType,
			EntryType:            domain.LedgerEntryTypeDebit,
			Amount:               event.Amount,
			Currency:             wallet.Currency,
			CounterpartyWalletID: event.CounterpartyWalletID,
			Reference:            event.Reference,
			Description:          event.Description,
			OccurredAt:           event.OccurredAt,
		}
		if err := h.repository.SaveLedgerEntriesTx(tx, entry); err != nil {
			return err
		}
		if err := validateTransferLedgerInvariant(tx, entry); err != nil {
			return err
		}
		logLedgerProjected(event, 1)
		return nil
	})
}

// applyIdempotent records the event in the processed_events table and then
// executes the projection inside a single DB transaction. If the event has
// already been processed, apply is never invoked and the transaction commits
// as a no-op.
func (h *WalletEventHandler) applyIdempotent(event corevents.BaseEvent, apply func(tx *gorm.DB) error) error {
	if event.GetEventID() == "" {
		return errors.New("event is missing event id — cannot apply idempotently")
	}
	observability.DefaultMetrics.ProjectionAttempts.Add(1)
	return h.repository.db.Transaction(func(tx *gorm.DB) error {
		record := &ProcessedEvent{
			EventID:     event.GetEventID(),
			EventType:   event.EventTypeName(),
			AggregateID: event.GetAggregateID(),
			Version:     event.GetVersion(),
			ProcessedAt: time.Now().UTC(),
		}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(record)
		if res.Error != nil {
			observability.DefaultMetrics.FailedEvents.Add(1)
			slog.Error("Projection inbox insert failed",
				"component", "projection",
				"eventId", event.GetEventID(),
				"eventType", event.EventTypeName(),
				"aggregateId", event.GetAggregateID(),
				"version", event.GetVersion(),
				"error", res.Error)
			return res.Error
		}
		if res.RowsAffected == 0 {
			slog.Info("Projection duplicate skipped",
				"component", "projection",
				"eventId", event.GetEventID(),
				"eventType", event.EventTypeName(),
				"aggregateId", event.GetAggregateID(),
				"version", event.GetVersion())
			observability.DefaultMetrics.SkippedEvents.Add(1)
			return nil
		}
		if err := apply(tx); err != nil {
			observability.DefaultMetrics.FailedEvents.Add(1)
			slog.Error("Projection failed",
				"component", "projection",
				"eventId", event.GetEventID(),
				"eventType", event.EventTypeName(),
				"aggregateId", event.GetAggregateID(),
				"version", event.GetVersion(),
				"error", err)
			return err
		}
		observability.DefaultMetrics.ProcessedEvents.Add(1)
		slog.Info("Projection applied",
			"component", "projection",
			"eventId", event.GetEventID(),
			"eventType", event.EventTypeName(),
			"aggregateId", event.GetAggregateID(),
			"version", event.GetVersion())
		return nil
	})
}

func findWalletInTx(tx *gorm.DB, id string) (*domain.Wallet, error) {
	var wallet domain.Wallet
	if err := tx.First(&wallet, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("wallet %s not found: %w", id, err)
	}
	return &wallet, nil
}

func transactionID(walletID string, version int) string {
	return fmt.Sprintf("%s-%d", walletID, version)
}

func ledgerEntryID(eventID, entryType string) string {
	return fmt.Sprintf("%s:%s", eventID, strings.ToLower(entryType))
}

func logLedgerProjected(event corevents.BaseEvent, entries int) {
	slog.Info("Ledger entries projected",
		"component", "ledger-projection",
		"eventId", event.GetEventID(),
		"eventType", event.EventTypeName(),
		"aggregateId", event.GetAggregateID(),
		"version", event.GetVersion(),
		"entries", entries)
}

func validateTransferLedgerInvariant(tx *gorm.DB, entry *domain.LedgerEntry) error {
	if entry == nil || !isTransferTransactionType(entry.TransactionType) {
		return nil
	}
	if entry.CounterpartyWalletID == "" || entry.Reference == "" {
		slog.Error("Ledger transfer invariant violated",
			"component", "ledger-projection",
			"eventId", entry.EventID,
			"eventType", entry.EventType,
			"walletId", entry.WalletID,
			"counterpartyWalletId", entry.CounterpartyWalletID,
			"reference", entry.Reference,
			"reason", "missing transfer counterpart metadata")
		return errors.New("ledger transfer invariant violated")
	}

	var counterpart domain.LedgerEntry
	err := tx.Where(
		"wallet_id = ? AND counterparty_wallet_id = ? AND reference = ? AND entry_type = ? AND occurred_at = ?",
		entry.CounterpartyWalletID,
		entry.WalletID,
		entry.Reference,
		inverseLedgerEntryType(entry.EntryType),
		entry.OccurredAt,
	).First(&counterpart).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if counterpart.Amount != entry.Amount || counterpart.Currency != entry.Currency {
		slog.Error("Ledger transfer invariant violated",
			"component", "ledger-projection",
			"eventId", entry.EventID,
			"eventType", entry.EventType,
			"walletId", entry.WalletID,
			"counterpartyWalletId", entry.CounterpartyWalletID,
			"reference", entry.Reference,
			"amount", entry.Amount,
			"counterpartAmount", counterpart.Amount,
			"currency", entry.Currency,
			"counterpartCurrency", counterpart.Currency,
			"reason", "debit and credit entries are not balanced")
		return errors.New("ledger transfer invariant violated")
	}
	return nil
}

func isTransferTransactionType(transactionType dto.TransactionType) bool {
	return transactionType == dto.TransactionTypeTransferIn || transactionType == dto.TransactionTypeTransferOut
}

func inverseLedgerEntryType(entryType string) string {
	if entryType == domain.LedgerEntryTypeDebit {
		return domain.LedgerEntryTypeCredit
	}
	return domain.LedgerEntryTypeDebit
}
