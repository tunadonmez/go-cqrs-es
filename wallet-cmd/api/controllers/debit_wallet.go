package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
)

// DebitWalletHandler handles PUT /api/v1/wallets/:id/debit.
func DebitWalletHandler(dispatcher *coreinfra.CommandDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cmd commands.DebitWalletCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: err.Error()})
			return
		}
		cmd.SetID(c.Param("id"))
		if err := dispatcher.Send(&cmd); err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.BaseResponse{Message: "Wallet debit request completed successfully!"})
	}
}
