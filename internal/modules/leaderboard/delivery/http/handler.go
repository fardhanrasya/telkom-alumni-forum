package http

import (
	"net/http"
	"strconv"

	leaderboardService "anoa.com/telkomalumiforum/internal/modules/leaderboard/service"
	"github.com/gin-gonic/gin"
)

type LeaderboardHandler struct {
	service leaderboardService.LeaderboardService
}

func NewLeaderboardHandler(service leaderboardService.LeaderboardService) *LeaderboardHandler {
	return &LeaderboardHandler{service: service}
}

func (h *LeaderboardHandler) GetLeaderboard(c *gin.Context) {
	timeframe := c.Query("timeframe") // "all_time", "monthly", "weekly"
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	leaderboard, err := h.service.GetLeaderboard(limit, timeframe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch leaderboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": leaderboard})
}
