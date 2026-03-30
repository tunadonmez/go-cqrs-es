package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/techbank/account-cmd/api/commands"
	"github.com/techbank/account-cmd/application"
	"github.com/techbank/account-common/dto"
)

type OpenAccountResponse struct {
	dto.BaseResponse
	ID string `json:"id"`
}

// OpenAccountHandler handles POST /api/v1/openBankAccount.
func OpenAccountHandler(handler *application.CommandHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cmd commands.OpenAccountCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: err.Error()})
			return
		}
		cmd.ID = uuid.New().String()
		if err := handler.HandleOpenAccount(&cmd); err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusCreated, OpenAccountResponse{
			BaseResponse: dto.BaseResponse{Message: "Bank account creation request completed successfully!"},
			ID:           cmd.ID,
		})
	}
}
