package handler

import (
	"net/http"

	"anoa.com/telkomalumiforum/internal/dto"
	"anoa.com/telkomalumiforum/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ThreadHandler struct {
	service service.ThreadService
}

func NewThreadHandler(service service.ThreadService) *ThreadHandler {
	return &ThreadHandler{service: service}
}

func (h *ThreadHandler) CreateThread(c *gin.Context) {
	var req dto.CreateThreadRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.service.CreateThread(c.Request.Context(), userID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "thread created successfully"})
}
