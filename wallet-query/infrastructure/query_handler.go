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
	q.Page = queries.NormalizePage(q.Page)
	q.PageSize = queries.NormalizePageSize(q.PageSize)
	q.SortBy, q.SortOrder = queries.NormalizeWalletSort(q.SortBy, q.SortOrder)
	wallets, err := h.repository.FindAllWallets(q)
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
	q.Page = queries.NormalizePage(q.Page)
	q.PageSize = queries.NormalizePageSize(q.PageSize)
	q.SortBy, q.SortOrder = queries.NormalizeTransactionSort(q.SortBy, q.SortOrder)
	transactions, err := h.repository.FindTransactionsByWalletID(q)
	if err != nil {
		return nil, err
	}
	result := make([]coredomain.BaseEntity, len(transactions))
	for i, transaction := range transactions {
		result[i] = transaction
	}
	return result, nil
}

func (h *WalletQueryHandler) HandleFindLedgerEntries(q queries.FindLedgerEntriesQuery) ([]coredomain.BaseEntity, error) {
	q.Page = queries.NormalizePage(q.Page)
	q.PageSize = queries.NormalizePageSize(q.PageSize)
	q.SortBy, q.SortOrder = queries.NormalizeLedgerSort(q.SortBy, q.SortOrder)
	entries, err := h.repository.FindLedgerEntries(q)
	if err != nil {
		return nil, err
	}
	result := make([]coredomain.BaseEntity, len(entries))
	for i, entry := range entries {
		result[i] = entry
	}
	return result, nil
}

func (h *WalletQueryHandler) HandleFindLedgerMovements(q queries.FindLedgerMovementsQuery) ([]coredomain.BaseEntity, error) {
	q.Page = queries.NormalizePage(q.Page)
	q.PageSize = queries.NormalizePageSize(q.PageSize)
	q.SortBy, q.SortOrder = queries.NormalizeLedgerMovementSort(q.SortBy, q.SortOrder)
	movements, err := h.repository.FindLedgerMovements(q)
	if err != nil {
		return nil, err
	}
	result := make([]coredomain.BaseEntity, len(movements))
	for i, movement := range movements {
		result[i] = movement
	}
	return result, nil
}

func (h *WalletQueryHandler) HandleFindLedgerMovementByID(q queries.FindLedgerMovementByIDQuery) ([]coredomain.BaseEntity, error) {
	movement, err := h.repository.FindLedgerMovementByID(q.ID)
	if err != nil {
		return nil, err
	}
	return []coredomain.BaseEntity{movement}, nil
}
