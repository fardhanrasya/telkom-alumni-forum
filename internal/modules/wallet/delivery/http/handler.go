package http

import (
	"net/http"
	"strconv"

	walletService "anoa.com/telkomalumiforum/internal/modules/wallet/service"
	"anoa.com/telkomalumiforum/pkg/response"
	"github.com/gin-gonic/gin"
)

type WalletHandler struct {
	service walletService.WalletService
}

func NewWalletHandler(service walletService.WalletService) *WalletHandler {
	return &WalletHandler{service: service}
}

func (h *WalletHandler) GetWallet(c *gin.Context) {
	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	wallet, err := h.service.GetWallet(c.Request.Context(), userID)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *WalletHandler) GetTransactions(c *gin.Context) {
	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	transactions, err := h.service.GetTransactions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	c.JSON(http.StatusOK, transactions)
}
