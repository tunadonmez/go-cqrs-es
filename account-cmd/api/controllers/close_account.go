package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tunadonmez/go-cqrs-es/account-cmd/api/commands"
	"github.com/tunadonmez/go-cqrs-es/account-common/dto"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
)

// CloseAccountHandler handles DELETE /api/v1/closeBankAccount/:id.
func CloseAccountHandler(dispatcher *coreinfra.CommandDispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		cmd := &commands.CloseAccountCommand{}
		cmd.SetID(c.Param("id"))
		if err := dispatcher.Send(cmd); err != nil {
			handleError(c, err)
			return
		}
		c.JSON(http.StatusOK, dto.BaseResponse{Message: "Bank account closure request successfully completed!"})
	}
}
