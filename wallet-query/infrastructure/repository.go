package infrastructure

import (
	"strings"
	"time"

	"github.com/tunadonmez/go-cqrs-es/wallet-query/api/queries"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WalletRepository provides PostgreSQL access for wallet read models.
type WalletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) SaveWallet(wallet *domain.Wallet) error {
	return r.db.Save(wallet).Error
}

func (r *WalletRepository) FindWalletByID(id string) (*domain.Wallet, error) {
	var wallet domain.Wallet
	if err := r.db.First(&wallet, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *WalletRepository) FindAllWallets(q queries.FindAllWalletsQuery) ([]*domain.Wallet, error) {
	var wallets []*domain.Wallet
	db := r.db.Model(&domain.Wallet{})
	if q.Currency != "" {
		db = db.Where("currency = ?", strings.ToUpper(strings.TrimSpace(q.Currency)))
	}
	orderBy := walletSortColumn(q.SortBy) + " " + q.SortOrder
	offset := (q.Page - 1) * q.PageSize
	if err := db.Order(orderBy).
		Limit(q.PageSize + 1).
		Offset(offset).
		Find(&wallets).Error; err != nil {
		return nil, err
	}
	return wallets, nil
}

func (r *WalletRepository) SaveTransaction(transaction *domain.Transaction) error {
	return r.db.Save(transaction).Error
}

func (r *WalletRepository) SaveLedgerEntriesTx(tx *gorm.DB, entries ...*domain.LedgerEntry) error {
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = time.Now().UTC()
		}
		if err := tx.Save(entry).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *WalletRepository) UpsertLedgerMovementTx(tx *gorm.DB, movement *domain.LedgerMovement) error {
	if movement == nil {
		return nil
	}
	now := time.Now().UTC()
	if movement.CreatedAt.IsZero() {
		movement.CreatedAt = now
	}
	movement.UpdatedAt = now
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "movement_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"movement_type":         movement.MovementType,
			"reference":             movement.Reference,
			"status":                movement.Status,
			"currency":              movement.Currency,
			"total_debit":           movement.TotalDebit,
			"total_credit":          movement.TotalCredit,
			"entry_count":           movement.EntryCount,
			"source_wallet_id":      movement.SourceWalletID,
			"destination_wallet_id": movement.DestinationWalletID,
			"aggregate_id":          movement.AggregateID,
			"event_id":              movement.EventID,
			"event_type":            movement.EventType,
			"occurred_at":           movement.OccurredAt,
			"updated_at":            movement.UpdatedAt,
		}),
	}).Create(movement).Error
}

func (r *WalletRepository) DeleteLedgerMovementTx(tx *gorm.DB, movementID string) error {
	if strings.TrimSpace(movementID) == "" {
		return nil
	}
	return tx.Delete(&domain.LedgerMovement{}, "movement_id = ?", movementID).Error
}

func (r *WalletRepository) FindLedgerMovementByID(id string) (*domain.LedgerMovement, error) {
	var movement domain.LedgerMovement
	if err := r.db.First(&movement, "movement_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &movement, nil
}

func (r *WalletRepository) FindLedgerEntriesByMovementIDTx(tx *gorm.DB, movementID string) ([]*domain.LedgerEntry, error) {
	var entries []*domain.LedgerEntry
	if err := tx.Where("movement_id = ?", movementID).
		Order("occurred_at asc").
		Order("id asc").
		Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *WalletRepository) FindAllLedgerMovementsForCheck() ([]*domain.LedgerMovement, error) {
	var movements []*domain.LedgerMovement
	if err := r.db.Order("occurred_at asc").Order("movement_id asc").Find(&movements).Error; err != nil {
		return nil, err
	}
	return movements, nil
}

func (r *WalletRepository) FindLedgerMovements(q queries.FindLedgerMovementsQuery) ([]*domain.LedgerMovement, error) {
	var movements []*domain.LedgerMovement
	db := r.db.Model(&domain.LedgerMovement{})
	if q.WalletID != "" {
		db = db.Where("source_wallet_id = ? OR destination_wallet_id = ?", q.WalletID, q.WalletID)
	}
	if q.MovementType != "" {
		db = db.Where("movement_type = ?", strings.ToUpper(strings.TrimSpace(q.MovementType)))
	}
	if q.Status != "" {
		db = db.Where("status = ?", strings.ToUpper(strings.TrimSpace(q.Status)))
	}
	if q.Reference != "" {
		db = db.Where("reference = ?", strings.TrimSpace(q.Reference))
	}
	if q.OccurredFrom != nil {
		db = db.Where("occurred_at >= ?", q.OccurredFrom.UTC())
	}
	if q.OccurredTo != nil {
		db = db.Where("occurred_at <= ?", q.OccurredTo.UTC())
	}
	orderBy := ledgerMovementSortColumn(q.SortBy) + " " + q.SortOrder
	offset := (q.Page - 1) * q.PageSize
	if err := db.Order(orderBy).
		Order("movement_id " + q.SortOrder).
		Limit(q.PageSize + 1).
		Offset(offset).
		Find(&movements).Error; err != nil {
		return nil, err
	}
	return movements, nil
}

func (r *WalletRepository) FindTransactionsByWalletID(q queries.FindWalletTransactionsQuery) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction
	db := r.db.Where("wallet_id = ?", q.WalletID)
	if q.Type != "" {
		db = db.Where("type = ?", q.Type)
	}
	if q.OccurredFrom != nil {
		db = db.Where("occurred_at >= ?", q.OccurredFrom.UTC())
	}
	if q.OccurredTo != nil {
		db = db.Where("occurred_at <= ?", q.OccurredTo.UTC())
	}
	orderBy := transactionSortColumn(q.SortBy) + " " + q.SortOrder
	offset := (q.Page - 1) * q.PageSize
	if err := db.Order(orderBy).
		Order("event_version " + q.SortOrder).
		Limit(q.PageSize + 1).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		return nil, err
	}
	return transactions, nil
}

func (r *WalletRepository) FindAllWalletsForCheck() ([]*domain.Wallet, error) {
	var wallets []*domain.Wallet
	if err := r.db.Order("id asc").Find(&wallets).Error; err != nil {
		return nil, err
	}
	return wallets, nil
}

func (r *WalletRepository) FindAllLedgerEntriesForCheck() ([]*domain.LedgerEntry, error) {
	var entries []*domain.LedgerEntry
	if err := r.db.Order("occurred_at asc").Order("id asc").Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *WalletRepository) FindLedgerEntries(q queries.FindLedgerEntriesQuery) ([]*domain.LedgerEntry, error) {
	var entries []*domain.LedgerEntry
	db := r.db.Model(&domain.LedgerEntry{})
	if q.MovementID != "" {
		db = db.Where("movement_id = ?", q.MovementID)
	}
	if q.WalletID != "" {
		db = db.Where("wallet_id = ?", q.WalletID)
	}
	if q.EntryType != "" {
		db = db.Where("entry_type = ?", strings.ToUpper(strings.TrimSpace(q.EntryType)))
	}
	if q.EventType != "" {
		db = db.Where("event_type = ?", strings.TrimSpace(q.EventType))
	}
	if q.OccurredFrom != nil {
		db = db.Where("occurred_at >= ?", q.OccurredFrom.UTC())
	}
	if q.OccurredTo != nil {
		db = db.Where("occurred_at <= ?", q.OccurredTo.UTC())
	}
	orderBy := ledgerSortColumn(q.SortBy) + " " + q.SortOrder
	offset := (q.Page - 1) * q.PageSize
	if err := db.Order(orderBy).
		Order("id " + q.SortOrder).
		Limit(q.PageSize + 1).
		Offset(offset).
		Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

func walletSortColumn(sortBy string) string {
	switch sortBy {
	case "balance":
		return "balance"
	case "owner":
		return "owner"
	default:
		return "created_at"
	}
}

func transactionSortColumn(sortBy string) string {
	switch sortBy {
	case "amount":
		return "amount"
	case "eventVersion":
		return "event_version"
	default:
		return "occurred_at"
	}
}

func ledgerSortColumn(sortBy string) string {
	switch sortBy {
	case "createdAt":
		return "created_at"
	default:
		return "occurred_at"
	}
}

func ledgerMovementSortColumn(sortBy string) string {
	switch sortBy {
	case "createdAt":
		return "created_at"
	default:
		return "occurred_at"
	}
}
