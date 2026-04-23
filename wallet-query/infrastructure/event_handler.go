package infrastructure

import (
	"errors"
	"fmt"
	"log/slog"
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
		return tx.Save(transaction).Error
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
		return tx.Save(transaction).Error
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
		return tx.Save(transaction).Error
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
