package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tunadonmez/go-cqrs-es/account-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/account-common/dto"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
)

type OpenAccountResponse struct {
	dto.BaseResponse
	ID string `json:"id"`
}

// OpenAccountHandler handles POST /api/v1/openBankAccount.
func OpenAccountHandler(dispatcher *coreinfra.CommandDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cmd commands.OpenAccountCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: err.Error()})
			return
		}
		cmd.SetID(uuid.New().String())
		if err := dispatcher.Send(&cmd); err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusCreated, OpenAccountResponse{
			BaseResponse: dto.BaseResponse{Message: "Bank account creation request completed successfully!"},
			ID:           cmd.GetID(),
		})
	}
}
