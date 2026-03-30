package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/techbank/account-cmd/api/commands"
	"github.com/techbank/account-cmd/application"
	"github.com/techbank/account-common/dto"
)

// WithdrawFundsHandler handles PUT /api/v1/withdrawFunds/:id.
func WithdrawFundsHandler(handler *application.CommandHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cmd commands.WithdrawFundsCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: err.Error()})
			return
		}
		cmd.ID = c.Param("id")
		if err := handler.HandleWithdrawFunds(&cmd); err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.BaseResponse{Message: "Withdraw funds request completed successfully!"})
	}
}
