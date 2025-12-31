package handler

import (
	"context"
	"fmt"
	"net/http"

	threadDto "anoa.com/telkomalumiforum/internal/modules/thread/dto"
	thread "anoa.com/telkomalumiforum/internal/modules/thread/service"
	commonDto "anoa.com/telkomalumiforum/pkg/dto"
	"anoa.com/telkomalumiforum/pkg/ratelimiter"
	"anoa.com/telkomalumiforum/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ThreadHandler struct {
	service thread.Service
}

func NewThreadHandler(service thread.Service) *ThreadHandler {
	return &ThreadHandler{service: service}
}

func (h *ThreadHandler) CreateThread(c *gin.Context) {
	var req threadDto.CreateThreadRequest
	if err := c.Bind(&req); err != nil {
		response.ResponseError(c, err)
		return
	}

	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	if err := h.service.CreateThread(c.Request.Context(), userID, req); err != nil {
		if rateLimitErr, ok := err.(*ratelimiter.RateLimitError); ok {
			c.Header("Retry-After", fmt.Sprintf("%.0f", rateLimitErr.RetryAfter.Seconds()))
			// Still handle specific rate limit error struct if not yet fully migrated to apperror everywhere
			// But ideally service returns ErrRateLimitExceeded wrap
			// Keeping backward compatibility or specific headers logic in handler
			c.JSON(http.StatusTooManyRequests, gin.H{"error": rateLimitErr.Message})
			return
		}
		response.ResponseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "thread created successfully"})
}

func (h *ThreadHandler) GetAllThreads(c *gin.Context) {
	var filter commonDto.ThreadFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.ResponseError(c, err) // ShouldBind errors are usually validation/bad request
		return
	}

	// Set defaults
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}

	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	threads, err := h.service.GetAllThreads(c.Request.Context(), userID, filter)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	c.JSON(http.StatusOK, threads)
}

func (h *ThreadHandler) GetMyThreads(c *gin.Context) {
	var filter commonDto.ThreadFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.ResponseError(c, err)
		return
	}

	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}

	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	threads, err := h.service.GetMyThreads(c.Request.Context(), userID, filter.Page, filter.Limit)
	if err != nil {
		response.ResponseError(c, err)
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

	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	if err := h.service.DeleteThread(c.Request.Context(), userID, threadID); err != nil {
		response.ResponseError(c, err)
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

	var req threadDto.UpdateThreadRequest
	if err := c.Bind(&req); err != nil {
		response.ResponseError(c, err)
		return
	}

	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	if err := h.service.UpdateThread(c.Request.Context(), userID, threadID, req); err != nil {
		response.ResponseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "thread updated successfully"})
}

func (h *ThreadHandler) GetThreadBySlug(c *gin.Context) {
	slug := c.Param("slug")

	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	// Get thread first
	thread, err := h.service.GetThreadBySlug(c.Request.Context(), userID, slug)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	// Increment view (async, in background)
	go func() {
		ctx := context.Background()
		_ = h.service.IncrementView(ctx, thread.ID, userID)
	}()

	c.JSON(http.StatusOK, thread)
}

func (h *ThreadHandler) GetThreadsByUsername(c *gin.Context) {
	username := c.Param("username")
	var filter commonDto.ThreadFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.ResponseError(c, err)
		return
	}

	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}

	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	threads, err := h.service.GetThreadsByUsername(c.Request.Context(), userID, username, filter.Page, filter.Limit)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	c.JSON(http.StatusOK, threads)
}
