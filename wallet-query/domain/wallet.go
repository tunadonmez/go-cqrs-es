package domain

import (
	"time"

	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
)

// Wallet is the PostgreSQL read-model entity.
type Wallet struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Owner     string    `json:"owner"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"createdAt"`
	Balance   float64   `json:"balance"`
}

func (w *Wallet) EntityID() string { return w.ID }

// Transaction is the projected wallet ledger entry for the read model.
type Transaction struct {
	ID                   string              `json:"id" gorm:"primaryKey"`
	WalletID             string              `json:"walletId" gorm:"index"`
	Type                 dto.TransactionType `json:"type"`
	Amount               float64             `json:"amount"`
	CounterpartyWalletID string              `json:"counterpartyWalletId,omitempty"`
	Reference            string              `json:"reference,omitempty"`
	Description          string              `json:"description,omitempty"`
	BalanceAfter         float64             `json:"balanceAfter"`
	OccurredAt           time.Time           `json:"occurredAt" gorm:"index"`
	EventVersion         int                 `json:"eventVersion"`
}

func (t *Transaction) EntityID() string { return t.ID }
