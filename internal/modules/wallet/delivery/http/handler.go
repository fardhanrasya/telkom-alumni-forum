package http

import (
	"net/http"

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
