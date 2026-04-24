package infrastructure

import (
	"crypto/sha1"
	"encoding/hex"
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
			MovementID:      movementIDForOpening(event),
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
		if event.OpeningBalance == 0 {
			entry.MovementID = ""
		}
		if err := h.repository.SaveLedgerEntriesTx(tx, entry); err != nil {
			return err
		}
		if entry.MovementID != "" {
			if err := h.projectLedgerMovementTx(tx, entry); err != nil {
				return err
			}
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
			MovementID:           movementIDForCredit(event, wallet.Currency),
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
		if entry.MovementID != "" {
			if err := h.projectLedgerMovementTx(tx, entry); err != nil {
				return err
			}
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
			MovementID:           movementIDForDebit(event, wallet.Currency),
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
		if entry.MovementID != "" {
			if err := h.projectLedgerMovementTx(tx, entry); err != nil {
				return err
			}
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

func (h *WalletEventHandler) projectLedgerMovementTx(tx *gorm.DB, entry *domain.LedgerEntry) error {
	if entry == nil || strings.TrimSpace(entry.MovementID) == "" {
		return nil
	}

	movement, foundEntries, err := h.buildLedgerMovementFromEntriesTx(tx, entry.MovementID, entry)
	if err != nil {
		return err
	}
	if len(foundEntries) == 0 {
		return h.repository.DeleteLedgerMovementTx(tx, entry.MovementID)
	}
	if err := h.repository.UpsertLedgerMovementTx(tx, movement); err != nil {
		return err
	}
	logLedgerMovementProjected(movement)
	return nil
}

func (h *WalletEventHandler) buildLedgerMovementFromEntriesTx(tx *gorm.DB, movementID string, trigger *domain.LedgerEntry) (*domain.LedgerMovement, []*domain.LedgerEntry, error) {
	entries, err := h.repository.FindLedgerEntriesByMovementIDTx(tx, movementID)
	if err != nil {
		return nil, nil, err
	}
	if len(entries) == 0 {
		return nil, nil, nil
	}

	movement := &domain.LedgerMovement{
		ID:           movementID,
		MovementType: ledgerMovementTypeFromEntry(entries[0]),
		Reference:    entries[0].Reference,
		OccurredAt:   entries[0].OccurredAt,
		AggregateID:  trigger.AggregateID,
		EventID:      trigger.EventID,
		EventType:    trigger.EventType,
		Currency:     entries[0].Currency,
		EntryCount:   len(entries),
	}

	currencies := map[string]struct{}{}
	debitCount := 0
	creditCount := 0
	sourceWallets := map[string]struct{}{}
	destinationWallets := map[string]struct{}{}
	status := domain.LedgerMovementStatusCompleted

	for _, item := range entries {
		currencies[item.Currency] = struct{}{}
		if item.OccurredAt.Before(movement.OccurredAt) {
			movement.OccurredAt = item.OccurredAt
		}
		if movement.Reference == "" && item.Reference != "" {
			movement.Reference = item.Reference
		}
		switch item.EntryType {
		case domain.LedgerEntryTypeDebit:
			movement.TotalDebit += item.Amount
			debitCount++
		case domain.LedgerEntryTypeCredit:
			movement.TotalCredit += item.Amount
			creditCount++
		}

		sourceWalletID, destinationWalletID := movementWalletsFromEntry(item)
		if sourceWalletID != "" {
			sourceWallets[sourceWalletID] = struct{}{}
			if movement.SourceWalletID == "" {
				movement.SourceWalletID = sourceWalletID
			}
		}
		if destinationWalletID != "" {
			destinationWallets[destinationWalletID] = struct{}{}
			if movement.DestinationWalletID == "" {
				movement.DestinationWalletID = destinationWalletID
			}
		}
	}

	if len(currencies) > 1 || len(sourceWallets) > 1 || len(destinationWallets) > 1 {
		status = domain.LedgerMovementStatusInconsistent
	}

	if movement.MovementType == domain.LedgerMovementTypeTransfer {
		switch {
		case len(entries) < 2:
			status = domain.LedgerMovementStatusPending
		case len(entries) != 2 || debitCount != 1 || creditCount != 1:
			status = domain.LedgerMovementStatusInconsistent
		case movement.TotalDebit != movement.TotalCredit:
			status = domain.LedgerMovementStatusInconsistent
		case movement.SourceWalletID == "" || movement.DestinationWalletID == "":
			status = domain.LedgerMovementStatusInconsistent
		}
	}

	movement.Status = status
	return movement, entries, nil
}

func logLedgerMovementProjected(movement *domain.LedgerMovement) {
	if movement == nil {
		return
	}
	message := "Ledger movement projected"
	if movement.MovementType == domain.LedgerMovementTypeTransfer {
		switch movement.Status {
		case domain.LedgerMovementStatusCompleted:
			message = "Transfer movement completed"
		case domain.LedgerMovementStatusPending:
			message = "Transfer movement updated"
		default:
			message = "Transfer movement inconsistent"
		}
	}
	slog.Info(message,
		"component", "ledger-projection",
		"movementId", movement.ID,
		"movementType", movement.MovementType,
		"status", movement.Status,
		"entryCount", movement.EntryCount,
		"totalDebit", movement.TotalDebit,
		"totalCredit", movement.TotalCredit,
		"sourceWalletId", movement.SourceWalletID,
		"destinationWalletId", movement.DestinationWalletID)
}

func validateTransferLedgerInvariant(tx *gorm.DB, entry *domain.LedgerEntry) error {
	if entry == nil || !isTransferTransactionType(entry.TransactionType) {
		return nil
	}
	if entry.CounterpartyWalletID == "" || entry.MovementID == "" {
		slog.Error("Ledger transfer invariant violated",
			"component", "ledger-projection",
			"eventId", entry.EventID,
			"eventType", entry.EventType,
			"walletId", entry.WalletID,
			"counterpartyWalletId", entry.CounterpartyWalletID,
			"movementId", entry.MovementID,
			"reason", "missing transfer movement metadata")
		return errors.New("ledger transfer invariant violated")
	}

	var counterpart domain.LedgerEntry
	err := tx.Where(
		"movement_id = ? AND wallet_id = ? AND counterparty_wallet_id = ? AND entry_type = ?",
		entry.MovementID,
		entry.CounterpartyWalletID,
		entry.WalletID,
		inverseLedgerEntryType(entry.EntryType),
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
			"movementId", entry.MovementID,
			"amount", entry.Amount,
			"counterpartAmount", counterpart.Amount,
			"currency", entry.Currency,
			"counterpartCurrency", counterpart.Currency,
			"reason", "debit and credit entries are not balanced")
		return errors.New("ledger transfer invariant violated")
	}
	return nil
}

func movementIDForOpening(event *commonevents.WalletCreatedEvent) string {
	return "opening:" + strings.TrimSpace(event.GetEventID())
}

func movementIDForCredit(event *commonevents.WalletCreditedEvent, currency string) string {
	if isTransferTransactionType(event.TransactionType) {
		return transferMovementID(event.GetAggregateID(), event.CounterpartyWalletID, event.Reference, event.OccurredAt, event.Amount, currency)
	}
	return "credit:" + strings.TrimSpace(event.GetEventID())
}

func movementIDForDebit(event *commonevents.WalletDebitedEvent, currency string) string {
	if isTransferTransactionType(event.TransactionType) {
		return transferMovementID(event.GetAggregateID(), event.CounterpartyWalletID, event.Reference, event.OccurredAt, event.Amount, currency)
	}
	return "debit:" + strings.TrimSpace(event.GetEventID())
}

func transferMovementID(walletID, counterpartyWalletID, reference string, occurredAt time.Time, amount float64, currency string) string {
	left, right := orderedPair(walletID, counterpartyWalletID)
	raw := fmt.Sprintf("%s|%s|%s|%s|%.8f|%s",
		left,
		right,
		normalizeMovementToken(reference),
		occurredAt.UTC().Format(time.RFC3339Nano),
		amount,
		strings.ToUpper(strings.TrimSpace(currency)),
	)
	sum := sha1.Sum([]byte(raw))
	return "transfer:" + hex.EncodeToString(sum[:])
}

func orderedPair(a, b string) (string, string) {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a <= b {
		return a, b
	}
	return b, a
}

func normalizeMovementToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "none"
	}
	return trimmed
}

func ledgerMovementTypeFromEntry(entry *domain.LedgerEntry) string {
	if entry == nil {
		return domain.LedgerMovementTypeCredit
	}
	switch entry.TransactionType {
	case dto.TransactionTypeOpeningBalance:
		return domain.LedgerMovementTypeOpeningBalance
	case dto.TransactionTypeTransferIn, dto.TransactionTypeTransferOut:
		return domain.LedgerMovementTypeTransfer
	case dto.TransactionTypeDebit:
		return domain.LedgerMovementTypeDebit
	default:
		return domain.LedgerMovementTypeCredit
	}
}

func movementWalletsFromEntry(entry *domain.LedgerEntry) (string, string) {
	if entry == nil {
		return "", ""
	}
	switch ledgerMovementTypeFromEntry(entry) {
	case domain.LedgerMovementTypeTransfer:
		if entry.EntryType == domain.LedgerEntryTypeDebit {
			return entry.WalletID, entry.CounterpartyWalletID
		}
		return entry.CounterpartyWalletID, entry.WalletID
	case domain.LedgerMovementTypeDebit:
		return entry.WalletID, ""
	default:
		return "", entry.WalletID
	}
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
