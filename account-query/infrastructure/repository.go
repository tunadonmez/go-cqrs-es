package infrastructure

import (
	"github.com/tunadonmez/go-cqrs-es/account-query/domain"
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

func (r *AccountRepository) DeleteByID(id string) error {
	return r.db.Delete(&domain.Account{}, "id = ?", id).Error
}
