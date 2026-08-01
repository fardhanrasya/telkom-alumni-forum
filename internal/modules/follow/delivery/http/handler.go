package http

import (
	"net/http"

	followService "anoa.com/telkomalumiforum/internal/modules/follow/service"
	"github.com/gin-gonic/gin"
)

type FollowHandler struct {
	followSvc followService.FollowService
}

func NewFollowHandler(followSvc followService.FollowService) *FollowHandler {
	return &FollowHandler{
		followSvc: followSvc,
	}
}

func (h *FollowHandler) ToggleFollow(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	targetUsername := c.Param("username")
	if targetUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
		return
	}

	isFollowing, err := h.followSvc.ToggleFollow(c.Request.Context(), userID.(string), targetUsername)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := "Berhasil mengikuti"
	if !isFollowing {
		message = "Berhasil berhenti mengikuti"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      message,
		"is_following": isFollowing,
	})
}

func (h *FollowHandler) GetFollowStatus(c *gin.Context) {
	targetUsername := c.Param("username")
	if targetUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
		return
	}

	currentUserIDStr := ""
	if val, exists := c.Get("user_id"); exists {
		if str, ok := val.(string); ok {
			currentUserIDStr = str
		}
	}

	status, err := h.followSvc.GetFollowStatus(c.Request.Context(), currentUserIDStr, targetUsername)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}
