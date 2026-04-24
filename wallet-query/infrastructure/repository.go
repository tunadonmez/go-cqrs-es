package infrastructure

import (
	"strings"
	"time"

	"github.com/tunadonmez/go-cqrs-es/wallet-query/api/queries"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/domain"
	"gorm.io/gorm"
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

func (r *WalletRepository) FindLedgerEntries(q queries.FindLedgerEntriesQuery) ([]*domain.LedgerEntry, error) {
	var entries []*domain.LedgerEntry
	db := r.db.Model(&domain.LedgerEntry{})
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
