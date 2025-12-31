package http

import (
	"net/http"
	"strconv"

	statService "anoa.com/telkomalumiforum/internal/modules/stat/service"
	"anoa.com/telkomalumiforum/internal/modules/thread/service"
	"github.com/gin-gonic/gin"
)

type StatHandler struct {
	statService   statService.StatService
	threadService thread.Service
}

func NewStatHandler(statService statService.StatService, threadService thread.Service) *StatHandler {
	return &StatHandler{
		statService:   statService,
		threadService: threadService,
	}
}

func (h *StatHandler) GetTotalUsers(c *gin.Context) {
	count, err := h.statService.GetTotalUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_users": count,
	})
}

func (h *StatHandler) GetTrendingThreads(c *gin.Context) {
	limitStr := c.Query("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	threads, err := h.threadService.GetTrendingThreads(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": threads,
	})
}
