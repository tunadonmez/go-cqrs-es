package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	coredomain "github.com/tunadonmez/go-cqrs-es/cqrs-core/domain"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/api/queries"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/domain"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/infrastructure"
)

type WalletListResponse struct {
	dto.BaseResponse
	Wallets    []*domain.Wallet `json:"wallets,omitempty"`
	Pagination *PaginationMeta  `json:"pagination,omitempty"`
}

type WalletDetailResponse struct {
	dto.BaseResponse
	Wallet *domain.Wallet `json:"wallet,omitempty"`
}

type WalletBalanceResponse struct {
	dto.BaseResponse
	WalletID string  `json:"walletId"`
	Currency string  `json:"currency"`
	Balance  float64 `json:"balance"`
}

type TransactionHistoryResponse struct {
	dto.BaseResponse
	Pagination   *PaginationMeta        `json:"pagination,omitempty"`
	Filters      *TransactionFilterMeta `json:"filters,omitempty"`
	Transactions []*domain.Transaction  `json:"transactions,omitempty"`
}

type PaginationMeta struct {
	Page          int    `json:"page"`
	PageSize      int    `json:"pageSize"`
	ReturnedItems int    `json:"returnedItems"`
	HasMore       bool   `json:"hasMore"`
	SortBy        string `json:"sortBy"`
	SortOrder     string `json:"sortOrder"`
}

type TransactionFilterMeta struct {
	Type         string `json:"type,omitempty"`
	OccurredFrom string `json:"occurredFrom,omitempty"`
	OccurredTo   string `json:"occurredTo,omitempty"`
}

type walletListParams struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
	SortBy    string `form:"sortBy"`
	SortOrder string `form:"sortOrder"`
	Currency  string `form:"currency"`
}

type walletTransactionsParams struct {
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
	SortBy       string `form:"sortBy"`
	SortOrder    string `form:"sortOrder"`
	Type         string `form:"type"`
	OccurredFrom string `form:"occurredFrom"`
	OccurredTo   string `form:"occurredTo"`
}

// RegisterRoutes wires wallet query handlers onto the given router group.
func RegisterRoutes(
	r *gin.RouterGroup,
	dispatcher *coreinfra.QueryDispatcher,
	deadLetters *infrastructure.DeadLetterRepository,
	deadLetterReprocessor *infrastructure.DeadLetterReprocessor,
) {
	r.GET("/wallets", getAllWallets(dispatcher))
	r.GET("/wallets/:id", getWalletByID(dispatcher))
	r.GET("/wallets/:id/balance", getWalletBalance(dispatcher))
	r.GET("/wallets/:id/transactions", getWalletTransactions(dispatcher))
	registerDeadLetterRoutes(r, deadLetters, deadLetterReprocessor)
}

func getAllWallets(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := walletListParams{}
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid wallet list query parameters!"})
			return
		}
		query := queries.FindAllWalletsQuery{
			Page:      params.Page,
			PageSize:  params.PageSize,
			SortBy:    params.SortBy,
			SortOrder: params.SortOrder,
			Currency:  params.Currency,
		}
		query.Page = queries.NormalizePage(query.Page)
		query.PageSize = queries.NormalizePageSize(query.PageSize)
		query.SortBy, query.SortOrder = queries.NormalizeWalletSort(query.SortBy, query.SortOrder)

		entities, err := dispatcher.Send(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, WalletListResponse{
				BaseResponse: dto.BaseResponse{Message: "Failed to complete get all wallets request!"},
			})
			return
		}
		if len(entities) == 0 {
			c.Status(http.StatusNoContent)
			return
		}
		wallets, hasMore := paginatedWallets(entities, query.PageSize)
		c.JSON(http.StatusOK, WalletListResponse{
			BaseResponse: dto.BaseResponse{Message: fmt.Sprintf("Successfully returned %d wallet(s)!", len(wallets))},
			Wallets:      wallets,
			Pagination: &PaginationMeta{
				Page:          query.Page,
				PageSize:      query.PageSize,
				ReturnedItems: len(wallets),
				HasMore:       hasMore,
				SortBy:        query.SortBy,
				SortOrder:     query.SortOrder,
			},
		})
	}
}

func getWalletByID(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entities, err := dispatcher.Send(queries.FindWalletByIDQuery{ID: id})
		if err != nil || len(entities) == 0 {
			c.Status(http.StatusNoContent)
			return
		}
		wallets := toWallets(entities)
		c.JSON(http.StatusOK, WalletDetailResponse{
			BaseResponse: dto.BaseResponse{Message: "Successfully returned wallet details!"},
			Wallet:       wallets[0],
		})
	}
}

func getWalletBalance(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entities, err := dispatcher.Send(queries.FindWalletByIDQuery{ID: id})
		if err != nil || len(entities) == 0 {
			c.Status(http.StatusNoContent)
			return
		}
		wallets := toWallets(entities)
		wallet := wallets[0]
		c.JSON(http.StatusOK, WalletBalanceResponse{
			BaseResponse: dto.BaseResponse{Message: "Successfully returned wallet balance!"},
			WalletID:     wallet.ID,
			Currency:     wallet.Currency,
			Balance:      wallet.Balance,
		})
	}
}

func getWalletTransactions(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		params := walletTransactionsParams{}
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid wallet transactions query parameters!"})
			return
		}
		occurredFrom, err := parseOptionalTime(params.OccurredFrom)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid occurredFrom query parameter! Use RFC3339 or YYYY-MM-DD."})
			return
		}
		occurredTo, err := parseOptionalTimeWithBounds(params.OccurredTo, true)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid occurredTo query parameter! Use RFC3339 or YYYY-MM-DD."})
			return
		}
		query := queries.FindWalletTransactionsQuery{
			WalletID:     id,
			Page:         params.Page,
			PageSize:     params.PageSize,
			SortBy:       params.SortBy,
			SortOrder:    params.SortOrder,
			OccurredFrom: occurredFrom,
			OccurredTo:   occurredTo,
		}
		if params.Type != "" {
			transactionType, ok := parseTransactionType(params.Type)
			if !ok {
				c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "Invalid type query parameter!"})
				return
			}
			query.Type = transactionType
		}
		if query.OccurredFrom != nil && query.OccurredTo != nil && query.OccurredFrom.After(*query.OccurredTo) {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: "occurredFrom must be before or equal to occurredTo!"})
			return
		}
		query.Page = queries.NormalizePage(query.Page)
		query.PageSize = queries.NormalizePageSize(query.PageSize)
		query.SortBy, query.SortOrder = queries.NormalizeTransactionSort(query.SortBy, query.SortOrder)

		entities, err := dispatcher.Send(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, TransactionHistoryResponse{
				BaseResponse: dto.BaseResponse{Message: "Failed to complete get wallet transactions request!"},
			})
			return
		}
		if len(entities) == 0 {
			c.Status(http.StatusNoContent)
			return
		}
		transactions, hasMore := paginatedTransactions(entities, query.PageSize)
		c.JSON(http.StatusOK, TransactionHistoryResponse{
			BaseResponse: dto.BaseResponse{Message: fmt.Sprintf("Successfully returned %d transaction(s)!", len(transactions))},
			Transactions: transactions,
			Pagination: &PaginationMeta{
				Page:          query.Page,
				PageSize:      query.PageSize,
				ReturnedItems: len(transactions),
				HasMore:       hasMore,
				SortBy:        query.SortBy,
				SortOrder:     query.SortOrder,
			},
			Filters: &TransactionFilterMeta{
				Type:         string(query.Type),
				OccurredFrom: formatOptionalTime(query.OccurredFrom),
				OccurredTo:   formatOptionalTime(query.OccurredTo),
			},
		})
	}
}

func toWallets(entities []coredomain.BaseEntity) []*domain.Wallet {
	result := make([]*domain.Wallet, 0, len(entities))
	for _, e := range entities {
		if wallet, ok := e.(*domain.Wallet); ok {
			result = append(result, wallet)
		}
	}
	return result
}

func toTransactions(entities []coredomain.BaseEntity) []*domain.Transaction {
	result := make([]*domain.Transaction, 0, len(entities))
	for _, e := range entities {
		if transaction, ok := e.(*domain.Transaction); ok {
			result = append(result, transaction)
		}
	}
	return result
}

func paginatedWallets(entities []coredomain.BaseEntity, pageSize int) ([]*domain.Wallet, bool) {
	wallets := toWallets(entities)
	if len(wallets) <= pageSize {
		return wallets, false
	}
	return wallets[:pageSize], true
}

func paginatedTransactions(entities []coredomain.BaseEntity, pageSize int) ([]*domain.Transaction, bool) {
	transactions := toTransactions(entities)
	if len(transactions) <= pageSize {
		return transactions, false
	}
	return transactions[:pageSize], true
}

func parseOptionalTime(raw string) (*time.Time, error) {
	return parseOptionalTimeWithBounds(raw, false)
}

func parseOptionalTimeWithBounds(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			if layout == "2006-01-02" {
				parsed = parsed.UTC()
				if endOfDay {
					parsed = parsed.Add(24*time.Hour - time.Nanosecond)
				}
			}
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("invalid time %q", raw)
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func parseTransactionType(raw string) (dto.TransactionType, bool) {
	transactionType := dto.TransactionType(strings.ToUpper(strings.TrimSpace(raw)))
	switch transactionType {
	case dto.TransactionTypeOpeningBalance,
		dto.TransactionTypeCredit,
		dto.TransactionTypeDebit,
		dto.TransactionTypeTransferIn,
		dto.TransactionTypeTransferOut:
		return transactionType, true
	default:
		return "", false
	}
}
