package domain

import (
	"time"

	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
)

const (
	LedgerEntryTypeDebit  = "DEBIT"
	LedgerEntryTypeCredit = "CREDIT"
)

// LedgerEntry is a read-side accounting view derived from projected events.
// It is not the source of truth; MongoDB events remain authoritative.
type LedgerEntry struct {
	ID                   string              `json:"id" gorm:"primaryKey"`
	WalletID             string              `json:"walletId" gorm:"column:wallet_id;index"`
	AggregateID          string              `json:"aggregateId" gorm:"column:aggregate_id;index"`
	TransactionID        string              `json:"transactionId" gorm:"column:transaction_id;index"`
	EventID              string              `json:"eventId" gorm:"column:event_id;index"`
	EventType            string              `json:"eventType" gorm:"column:event_type;index"`
	EventVersion         int                 `json:"eventVersion" gorm:"column:event_version"`
	TransactionType      dto.TransactionType `json:"transactionType" gorm:"column:transaction_type;index"`
	EntryType            string              `json:"entryType" gorm:"column:entry_type;index"`
	Amount               float64             `json:"amount"`
	Currency             string              `json:"currency" gorm:"index"`
	CounterpartyWalletID string              `json:"counterpartyWalletId,omitempty" gorm:"column:counterparty_wallet_id;index"`
	Reference            string              `json:"reference,omitempty" gorm:"index"`
	Description          string              `json:"description,omitempty"`
	OccurredAt           time.Time           `json:"occurredAt" gorm:"column:occurred_at;index"`
	CreatedAt            time.Time           `json:"createdAt" gorm:"column:created_at;index"`
}

func (l *LedgerEntry) EntityID() string { return l.ID }

func (LedgerEntry) TableName() string { return "ledger_entries" }
