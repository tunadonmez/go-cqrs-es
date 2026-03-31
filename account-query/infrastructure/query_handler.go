package infrastructure

import (
	"github.com/tunadonmez/go-cqrs-es/account-query/api/queries"
	coredomain "github.com/tunadonmez/go-cqrs-es/cqrs-core/domain"
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
