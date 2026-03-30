package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/techbank/account-cmd/infrastructure"
	"github.com/techbank/account-common/dto"
)

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, infrastructure.ErrConcurrency):
		c.JSON(http.StatusConflict, dto.BaseResponse{Message: err.Error()})
	case errors.Is(err, infrastructure.ErrAggregateNotFound):
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: err.Error()})
	default:
		// Business rule violations (IllegalStateException equivalent)
		c.JSON(http.StatusBadRequest, dto.BaseResponse{Message: err.Error()})
	}
}
