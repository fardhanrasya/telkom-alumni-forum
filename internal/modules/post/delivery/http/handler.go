package handler

import (
	"fmt"
	"net/http"

	postDto "anoa.com/telkomalumiforum/internal/modules/post/dto"
	post "anoa.com/telkomalumiforum/internal/modules/post/service"
	"anoa.com/telkomalumiforum/pkg/ratelimiter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PostHandler struct {
	service post.PostService
}

func NewPostHandler(service post.PostService) *PostHandler {
	return &PostHandler{service: service}
}

func (h *PostHandler) CreatePost(c *gin.Context) {
	threadIDStr := c.Param("thread_id")
	// Note: The param is in URL but also expected in body by DTO logic?
	// Actually DTO has ThreadID string.
	// We should probably override it with URL param or validate they match.
	// Simplest: bind JSON, then overwrite ThreadID from Param.

	var req postDto.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ThreadID = threadIDStr

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

	resp, err := h.service.CreatePost(c.Request.Context(), userID, req)
	if err != nil {
		if rateLimitErr, ok := err.(*ratelimiter.RateLimitError); ok {
			c.Header("Retry-After", fmt.Sprintf("%.0f", rateLimitErr.RetryAfter.Seconds()))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": rateLimitErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *PostHandler) GetPostsByThreadID(c *gin.Context) {
	threadIDStr := c.Param("thread_id")
	threadID, err := uuid.Parse(threadIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thread id"})
		return
	}

	var filter postDto.PostFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	posts, err := h.service.GetPostsByThreadID(c.Request.Context(), threadID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, posts)
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	postIDStr := c.Param("post_id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	var req postDto.UpdatePostRequest
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

	resp, err := h.service.UpdatePost(c.Request.Context(), userID, postID, req)
	if err != nil {
		// Differentiate errors (forbidden vs internal)
		if err.Error() == "unauthorized: you can only update your own post" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	postIDStr := c.Param("post_id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, _ := uuid.Parse(userIDStr.(string))

	if err := h.service.DeletePost(c.Request.Context(), userID, postID); err != nil {
		if err.Error() == "unauthorized: you can only delete your own post unless you are an admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "post deleted successfully"})
}

func (h *PostHandler) GetPostByID(c *gin.Context) {
	postIDStr := c.Param("post_id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	post, err := h.service.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, post)
}
