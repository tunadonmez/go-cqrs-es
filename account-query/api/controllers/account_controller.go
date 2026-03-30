package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/techbank/account-common/dto"
	"github.com/techbank/account-query/api/queries"
	"github.com/techbank/account-query/domain"
	"github.com/techbank/account-query/infrastructure"
	coredomain "github.com/techbank/cqrs-core/domain"
)

// AccountResponse wraps a list of accounts in the standard response envelope.
type AccountResponse struct {
	dto.BaseResponse
	Accounts []*domain.Account `json:"accounts,omitempty"`
}

// RegisterRoutes wires account query handlers onto the given router group.
func RegisterRoutes(r *gin.RouterGroup, qh *infrastructure.AccountQueryHandler) {
	r.GET("/accounts", getAllAccounts(qh))
	r.GET("/accounts/:id", getAccountByID(qh))
}

func getAllAccounts(qh *infrastructure.AccountQueryHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		entities, err := qh.HandleFindAll(queries.FindAllAccountsQuery{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, AccountResponse{
				BaseResponse: dto.BaseResponse{Message: "Failed to complete get all accounts request!"},
			})
			return
		}
		if len(entities) == 0 {
			c.Status(http.StatusNoContent)
			return
		}
		accounts := toAccounts(entities)
		c.JSON(http.StatusOK, AccountResponse{
			BaseResponse: dto.BaseResponse{Message: fmt.Sprintf("Successfully returned %d account(s)!", len(accounts))},
			Accounts:     accounts,
		})
	}
}

func getAccountByID(qh *infrastructure.AccountQueryHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entities, err := qh.HandleFindByID(queries.FindAccountByIdQuery{ID: id})
		if err != nil || len(entities) == 0 {
			c.Status(http.StatusNoContent)
			return
		}
		accounts := toAccounts(entities)
		c.JSON(http.StatusOK, AccountResponse{
			BaseResponse: dto.BaseResponse{Message: "Successfully returned account!"},
			Accounts:     accounts,
		})
	}
}

func toAccounts(entities []coredomain.BaseEntity) []*domain.Account {
	result := make([]*domain.Account, 0, len(entities))
	for _, e := range entities {
		if a, ok := e.(*domain.Account); ok {
			result = append(result, a)
		}
	}
	return result
}
