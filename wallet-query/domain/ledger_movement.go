package domain

import "time"

const (
	LedgerMovementTypeOpeningBalance = "OPENING_BALANCE"
	LedgerMovementTypeCredit         = "CREDIT"
	LedgerMovementTypeDebit          = "DEBIT"
	LedgerMovementTypeTransfer       = "TRANSFER"

	LedgerMovementStatusPending      = "PENDING"
	LedgerMovementStatusCompleted    = "COMPLETED"
	LedgerMovementStatusInconsistent = "INCONSISTENT"
)

// LedgerMovement is a projected journal/movement summary derived from
// ledger_entries. It is an audit-oriented read model, not source of truth.
type LedgerMovement struct {
	ID                  string    `json:"id" gorm:"primaryKey;column:movement_id"`
	MovementType        string    `json:"movementType" gorm:"column:movement_type;index"`
	Reference           string    `json:"reference,omitempty" gorm:"index"`
	Status              string    `json:"status" gorm:"index"`
	Currency            string    `json:"currency" gorm:"index"`
	TotalDebit          float64   `json:"totalDebit" gorm:"column:total_debit"`
	TotalCredit         float64   `json:"totalCredit" gorm:"column:total_credit"`
	EntryCount          int       `json:"entryCount" gorm:"column:entry_count"`
	SourceWalletID      string    `json:"sourceWalletId,omitempty" gorm:"column:source_wallet_id;index"`
	DestinationWalletID string    `json:"destinationWalletId,omitempty" gorm:"column:destination_wallet_id;index"`
	AggregateID         string    `json:"aggregateId,omitempty" gorm:"column:aggregate_id;index"`
	EventID             string    `json:"eventId,omitempty" gorm:"column:event_id;index"`
	EventType           string    `json:"eventType,omitempty" gorm:"column:event_type;index"`
	OccurredAt          time.Time `json:"occurredAt" gorm:"column:occurred_at;index"`
	CreatedAt           time.Time `json:"createdAt" gorm:"column:created_at;index"`
	UpdatedAt           time.Time `json:"updatedAt" gorm:"column:updated_at;index"`
}

func (l *LedgerMovement) EntityID() string { return l.ID }

func (LedgerMovement) TableName() string { return "ledger_movements" }
