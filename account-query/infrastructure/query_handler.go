package infrastructure

import (
	"github.com/techbank/account-query/api/queries"
	coredomain "github.com/techbank/cqrs-core/domain"
)

// AccountQueryHandler handles all account queries against the read model.
type AccountQueryHandler struct {
	repository *AccountRepository
}

func NewAccountQueryHandler(repo *AccountRepository) *AccountQueryHandler {
	return &AccountQueryHandler{repository: repo}
}

func (h *AccountQueryHandler) HandleFindAll(q queries.FindAllAccountsQuery) ([]coredomain.BaseEntity, error) {
	accounts, err := h.repository.FindAll()
	if err != nil {
		return nil, err
	}
	result := make([]coredomain.BaseEntity, len(accounts))
	for i, a := range accounts {
		result[i] = a
	}
	return result, nil
}

func (h *AccountQueryHandler) HandleFindByID(q queries.FindAccountByIdQuery) ([]coredomain.BaseEntity, error) {
	account, err := h.repository.FindByID(q.ID)
	if err != nil {
		return nil, err
	}
	return []coredomain.BaseEntity{account}, nil
}

func (h *AccountQueryHandler) HandleFindByHolder(q queries.FindAccountByHolderQuery) ([]coredomain.BaseEntity, error) {
	account, err := h.repository.FindByAccountHolder(q.AccountHolder)
	if err != nil {
		return nil, err
	}
	return []coredomain.BaseEntity{account}, nil
}

func (h *AccountQueryHandler) HandleFindByBalance(q queries.FindAccountWithBalanceQuery) ([]coredomain.BaseEntity, error) {
	if q.EqualityType == queries.GreaterThan {
		return h.repository.FindByBalanceGreaterThan(q.Balance)
	}
	return h.repository.FindByBalanceLessThan(q.Balance)
}
