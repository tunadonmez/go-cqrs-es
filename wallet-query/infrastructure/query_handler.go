package infrastructure

import (
	coredomain "github.com/tunadonmez/go-cqrs-es/cqrs-core/domain"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/api/queries"
)

// WalletQueryHandler handles all wallet queries against the read model.
type WalletQueryHandler struct {
	repository *WalletRepository
}

func NewWalletQueryHandler(repo *WalletRepository) *WalletQueryHandler {
	return &WalletQueryHandler{repository: repo}
}

func (h *WalletQueryHandler) HandleFindAll(q queries.FindAllWalletsQuery) ([]coredomain.BaseEntity, error) {
	wallets, err := h.repository.FindAllWallets()
	if err != nil {
		return nil, err
	}
	result := make([]coredomain.BaseEntity, len(wallets))
	for i, wallet := range wallets {
		result[i] = wallet
	}
	return result, nil
}

func (h *WalletQueryHandler) HandleFindByID(q queries.FindWalletByIDQuery) ([]coredomain.BaseEntity, error) {
	wallet, err := h.repository.FindWalletByID(q.ID)
	if err != nil {
		return nil, err
	}
	return []coredomain.BaseEntity{wallet}, nil
}

func (h *WalletQueryHandler) HandleFindTransactions(q queries.FindWalletTransactionsQuery) ([]coredomain.BaseEntity, error) {
	transactions, err := h.repository.FindTransactionsByWalletID(q.WalletID)
	if err != nil {
		return nil, err
	}
	result := make([]coredomain.BaseEntity, len(transactions))
	for i, transaction := range transactions {
		result[i] = transaction
	}
	return result, nil
}
