package infrastructure

import (
	"github.com/techbank/account-query/domain"
	coredomain "github.com/techbank/cqrs-core/domain"
	"gorm.io/gorm"
)

// AccountRepository provides MySQL access for the Account read model.
type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Save(account *domain.Account) error {
	return r.db.Save(account).Error
}

func (r *AccountRepository) FindByID(id string) (*domain.Account, error) {
	var account domain.Account
	if err := r.db.First(&account, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *AccountRepository) FindAll() ([]*domain.Account, error) {
	var accounts []*domain.Account
	if err := r.db.Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *AccountRepository) FindByAccountHolder(holder string) (*domain.Account, error) {
	var account domain.Account
	if err := r.db.First(&account, "account_holder = ?", holder).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *AccountRepository) FindByBalanceGreaterThan(balance float64) ([]coredomain.BaseEntity, error) {
	var accounts []*domain.Account
	if err := r.db.Where("balance > ?", balance).Find(&accounts).Error; err != nil {
		return nil, err
	}
	result := make([]coredomain.BaseEntity, len(accounts))
	for i, a := range accounts {
		result[i] = a
	}
	return result, nil
}

func (r *AccountRepository) FindByBalanceLessThan(balance float64) ([]coredomain.BaseEntity, error) {
	var accounts []*domain.Account
	if err := r.db.Where("balance < ?", balance).Find(&accounts).Error; err != nil {
		return nil, err
	}
	result := make([]coredomain.BaseEntity, len(accounts))
	for i, a := range accounts {
		result[i] = a
	}
	return result, nil
}

func (r *AccountRepository) DeleteByID(id string) error {
	return r.db.Delete(&domain.Account{}, "id = ?", id).Error
}
