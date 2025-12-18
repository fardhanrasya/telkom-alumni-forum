package handler

import (
	"context"
	"fmt"
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
		if rateLimitErr, ok := err.(*service.RateLimitError); ok {
			c.Header("Retry-After", fmt.Sprintf("%.0f", rateLimitErr.RetryAfter.Seconds()))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": rateLimitErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "thread created successfully"})
}

func (h *ThreadHandler) GetAllThreads(c *gin.Context) {
	var filter dto.ThreadFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
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

	threads, err := h.service.GetAllThreads(c.Request.Context(), userID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, threads)
}

func (h *ThreadHandler) GetMyThreads(c *gin.Context) {
	var filter dto.ThreadFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
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

	threads, err := h.service.GetMyThreads(c.Request.Context(), userID, filter.Page, filter.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, threads)
}

func (h *ThreadHandler) DeleteThread(c *gin.Context) {
	idStr := c.Param("thread_id")
	threadID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thread id"})
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

	if err := h.service.DeleteThread(c.Request.Context(), userID, threadID); err != nil {
		// Basic error string matching, ideally should use custom errors or checks
		if err.Error() == "unauthorized: you can only delete your own threads unless you are an admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "thread deleted successfully"})
}

func (h *ThreadHandler) UpdateThread(c *gin.Context) {
	idStr := c.Param("thread_id")
	threadID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thread id"})
		return
	}

	var req dto.UpdateThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, _ := uuid.Parse(userIDStr.(string))

	if err := h.service.UpdateThread(c.Request.Context(), userID, threadID, req); err != nil {
		if err.Error() == "unauthorized: you can only update your own thread" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "thread updated successfully"})
}

func (h *ThreadHandler) GetThreadBySlug(c *gin.Context) {
	slug := c.Param("slug")

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

	// Get thread first
	thread, err := h.service.GetThreadBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"thread not found": err.Error()})
		return
	}

	// Increment view (async, in background)
	go func() {
		ctx := context.Background()
		_ = h.service.IncrementView(ctx, thread.ID, userID)
	}()

	c.JSON(http.StatusOK, thread)
}
