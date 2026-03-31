package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tunadonmez/go-cqrs-es/account-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/account-common/dto"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
)

// DepositFundsHandler handles PUT /api/v1/depositFunds/:id.
func DepositFundsHandler(dispatcher *coreinfra.CommandDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cmd commands.DepositFundsCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: err.Error()})
			return
		}
		cmd.SetID(c.Param("id"))
		if err := dispatcher.Send(&cmd); err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.BaseResponse{Message: "Deposit funds request completed successfully!"})
	}
}
