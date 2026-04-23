package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
)

type CreateWalletResponse struct {
	dto.BaseResponse
	ID string `json:"id"`
}

// CreateWalletHandler handles POST /api/v1/wallets.
func CreateWalletHandler(dispatcher *coreinfra.CommandDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cmd commands.CreateWalletCommand
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: err.Error()})
			return
		}
		cmd.SetID(uuid.New().String())
		if err := dispatcher.Send(&cmd); err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusCreated, CreateWalletResponse{
			BaseResponse: dto.BaseResponse{Message: "Wallet creation request completed successfully!"},
			ID:           cmd.GetID(),
		})
	}
}
