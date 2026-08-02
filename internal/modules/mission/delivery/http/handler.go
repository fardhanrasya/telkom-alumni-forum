package http

import (
	"errors"
	"net/http"
	"strconv"

	missionService "anoa.com/telkomalumiforum/internal/modules/mission/service"
	walletRepo "anoa.com/telkomalumiforum/internal/modules/wallet/repository"
	"anoa.com/telkomalumiforum/pkg/response"
	"github.com/gin-gonic/gin"
)

type MissionHandler struct {
	service missionService.MissionService
}

func NewMissionHandler(service missionService.MissionService) *MissionHandler {
	return &MissionHandler{service: service}
}

func (h *MissionHandler) GetMissions(c *gin.Context) {
	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	missions, err := h.service.GetMissions(c.Request.Context(), userID)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": missions})
}

func (h *MissionHandler) ClaimMission(c *gin.Context) {
	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	missionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id misi tidak valid"})
		return
	}

	result, err := h.service.Claim(c.Request.Context(), userID, uint(missionID))
	if err != nil {
		switch {
		case errors.Is(err, missionService.ErrMissionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "misi_tidak_ditemukan", "message": err.Error()})
		case errors.Is(err, missionService.ErrNotClaimable):
			c.JSON(http.StatusBadRequest, gin.H{"error": "misi_belum_siap", "message": err.Error()})
		case errors.Is(err, missionService.ErrAlreadyClaimed):
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah_diklaim", "message": err.Error()})
		case errors.Is(err, walletRepo.ErrInsufficientBalance):
			// Unreachable for a credit, kept for completeness/symmetry with purchase errors.
			c.JSON(http.StatusBadRequest, gin.H{"error": "saldo_tidak_cukup", "message": err.Error()})
		default:
			response.ResponseError(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, result)
}
