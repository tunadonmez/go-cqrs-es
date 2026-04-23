package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/infrastructure"
	"github.com/tunadonmez/go-cqrs-es/wallet-common/dto"
)

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, infrastructure.ErrConcurrency):
		c.JSON(http.StatusConflict, dto.BaseResponse{Message: err.Error()})
	case errors.Is(err, infrastructure.ErrWalletNotFound):
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: err.Error()})
	default:
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: err.Error()})
	}
}
