package infrastructure

import (
	"fmt"

	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
	commonevents "github.com/tunadonmez/go-cqrs-es/wallet-common/events"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/domain"
	"gorm.io/gorm"
)

// WalletEventHandler projects domain events onto PostgreSQL read models.
type WalletEventHandler struct {
	repository *WalletRepository
}

func NewWalletEventHandler(repo *WalletRepository) *WalletEventHandler {
	return &WalletEventHandler{repository: repo}
}

func (h *WalletEventHandler) OnWalletCreated(event *commonevents.WalletCreatedEvent) error {
	wallet := &domain.Wallet{
		ID:        event.GetID(),
		Owner:     event.Owner,
		Currency:  event.Currency,
		CreatedAt: event.CreatedAt,
		Balance:   event.OpeningBalance,
	}

	transaction := &domain.Transaction{
		ID:           transactionID(event.GetID(), event.GetVersion()),
		WalletID:     event.GetID(),
		Type:         dto.TransactionTypeOpeningBalance,
		Amount:       event.OpeningBalance,
		Description:  "wallet created",
		BalanceAfter: event.OpeningBalance,
		OccurredAt:   event.CreatedAt,
		EventVersion: event.GetVersion(),
	}

	return h.repository.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(wallet).Error; err != nil {
			return err
		}
		return tx.Save(transaction).Error
	})
}

func (h *WalletEventHandler) OnWalletCredited(event *commonevents.WalletCreditedEvent) error {
	wallet, err := h.repository.FindWalletByID(event.GetID())
	if err != nil {
		return nil
	}
	wallet.Balance += event.Amount

	transaction := &domain.Transaction{
		ID:                   transactionID(event.GetID(), event.GetVersion()),
		WalletID:             event.GetID(),
		Type:                 event.TransactionType,
		Amount:               event.Amount,
		CounterpartyWalletID: event.CounterpartyWalletID,
		Reference:            event.Reference,
		Description:          event.Description,
		BalanceAfter:         wallet.Balance,
		OccurredAt:           event.OccurredAt,
		EventVersion:         event.GetVersion(),
	}

	return h.repository.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(wallet).Error; err != nil {
			return err
		}
		return tx.Save(transaction).Error
	})
}

func (h *WalletEventHandler) OnWalletDebited(event *commonevents.WalletDebitedEvent) error {
	wallet, err := h.repository.FindWalletByID(event.GetID())
	if err != nil {
		return nil
	}
	wallet.Balance -= event.Amount

	transaction := &domain.Transaction{
		ID:                   transactionID(event.GetID(), event.GetVersion()),
		WalletID:             event.GetID(),
		Type:                 event.TransactionType,
		Amount:               event.Amount,
		CounterpartyWalletID: event.CounterpartyWalletID,
		Reference:            event.Reference,
		Description:          event.Description,
		BalanceAfter:         wallet.Balance,
		OccurredAt:           event.OccurredAt,
		EventVersion:         event.GetVersion(),
	}

	return h.repository.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(wallet).Error; err != nil {
			return err
		}
		return tx.Save(transaction).Error
	})
}

func transactionID(walletID string, version int) string {
	return fmt.Sprintf("%s-%d", walletID, version)
}
