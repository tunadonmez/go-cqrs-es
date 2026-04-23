package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	coredomain "github.com/tunadonmez/go-cqrs-es/cqrs-core/domain"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/api/queries"
	"github.com/tunadonmez/go-cqrs-es/wallet-query/domain"
)

type WalletListResponse struct {
	dto.BaseResponse
	Wallets []*domain.Wallet `json:"wallets,omitempty"`
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
	Transactions []*domain.Transaction `json:"transactions,omitempty"`
}

// RegisterRoutes wires wallet query handlers onto the given router group.
func RegisterRoutes(r *gin.RouterGroup, dispatcher *coreinfra.QueryDispatcher) {
	r.GET("/wallets", getAllWallets(dispatcher))
	r.GET("/wallets/:id", getWalletByID(dispatcher))
	r.GET("/wallets/:id/balance", getWalletBalance(dispatcher))
	r.GET("/wallets/:id/transactions", getWalletTransactions(dispatcher))
}

func getAllWallets(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		entities, err := dispatcher.Send(queries.FindAllWalletsQuery{})
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
		wallets := toWallets(entities)
		c.JSON(http.StatusOK, WalletListResponse{
			BaseResponse: dto.BaseResponse{Message: fmt.Sprintf("Successfully returned %d wallet(s)!", len(wallets))},
			Wallets:      wallets,
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
		entities, err := dispatcher.Send(queries.FindWalletTransactionsQuery{WalletID: id})
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
		transactions := toTransactions(entities)
		c.JSON(http.StatusOK, TransactionHistoryResponse{
			BaseResponse: dto.BaseResponse{Message: fmt.Sprintf("Successfully returned %d transaction(s)!", len(transactions))},
			Transactions: transactions,
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
