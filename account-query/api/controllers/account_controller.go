package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tunadonmez/go-cqrs-es/account-common/dto"
	"github.com/tunadonmez/go-cqrs-es/account-query/api/queries"
	"github.com/tunadonmez/go-cqrs-es/account-query/domain"
	coredomain "github.com/tunadonmez/go-cqrs-es/cqrs-core/domain"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
)

// AccountResponse wraps a list of accounts in the standard response envelope.
type AccountResponse struct {
	dto.BaseResponse
	Accounts []*domain.Account `json:"accounts,omitempty"`
}

// RegisterRoutes wires account query handlers onto the given router group.
func RegisterRoutes(r *gin.RouterGroup, dispatcher *coreinfra.QueryDispatcher) {
	r.GET("/accounts", getAllAccounts(dispatcher))
	r.GET("/accounts/:id", getAccountByID(dispatcher))
}

func getAllAccounts(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		entities, err := dispatcher.Send(queries.FindAllAccountsQuery{})
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

func getAccountByID(dispatcher *coreinfra.QueryDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		entities, err := dispatcher.Send(queries.FindAccountByIdQuery{ID: id})
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
