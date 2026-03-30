package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/techbank/account-cmd/api/commands"
	"github.com/techbank/account-cmd/application"
	"github.com/techbank/account-common/dto"
)

// CloseAccountHandler handles DELETE /api/v1/closeBankAccount/:id.
func CloseAccountHandler(handler *application.CommandHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		cmd := &commands.CloseAccountCommand{}
		cmd.ID = c.Param("id")
		if err := handler.HandleCloseAccount(cmd); err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.BaseResponse{Message: "Bank account closure request successfully completed!"})
	}
}
