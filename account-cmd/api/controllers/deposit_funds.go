package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/techbank/account-cmd/api/commands"
	"github.com/techbank/account-cmd/application"
	"github.com/techbank/account-common/dto"
)

// DepositFundsHandler handles PUT /api/v1/depositFunds/:id.
func DepositFundsHandler(handler *application.CommandHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cmd commands.DepositFundsCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: err.Error()})
			return
		}
		cmd.ID = c.Param("id")
		if err := handler.HandleDepositFunds(&cmd); err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.BaseResponse{Message: "Deposit funds request completed successfully!"})
	}
}
