package infrastructure

import (
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

func (r *WalletRepository) FindAllWallets() ([]*domain.Wallet, error) {
	var wallets []*domain.Wallet
	if err := r.db.Order("created_at asc").Find(&wallets).Error; err != nil {
		return nil, err
	}
	return wallets, nil
}

func (r *WalletRepository) SaveTransaction(transaction *domain.Transaction) error {
	return r.db.Save(transaction).Error
}

func (r *WalletRepository) FindTransactionsByWalletID(walletID string) ([]*domain.Transaction, error) {
	var transactions []*domain.Transaction
	if err := r.db.Where("wallet_id = ?", walletID).
		Order("occurred_at asc").
		Order("event_version asc").
		Find(&transactions).Error; err != nil {
		return nil, err
	}
	return transactions, nil
}
