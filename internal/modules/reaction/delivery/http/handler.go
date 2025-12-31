package handler

import (
	"net/http"

	reactionDto "anoa.com/telkomalumiforum/internal/modules/reaction/dto"
	reaction "anoa.com/telkomalumiforum/internal/modules/reaction/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReactionHandler struct {
	service reaction.ReactionService
}

func NewReactionHandler(service reaction.ReactionService) *ReactionHandler {
	return &ReactionHandler{service: service}
}

func (h *ReactionHandler) ToggleReaction(c *gin.Context) {
	var req reactionDto.ReactionToggleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	if err := h.service.ToggleReaction(c.Request.Context(), userID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

func (h *ReactionHandler) GetReactions(c *gin.Context) {
	refType := c.Param("refType")
	refIDStr := c.Param("refID")

	refID, err := uuid.Parse(refIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reference id"})
		return
	}

	// Validate refType
	if refType != "thread" && refType != "post" && refType != "menfess" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reference type"})
		return
	}

	var userIDPtr *uuid.UUID
	// Optional User ID (for checking "user_reacted")
	userIDStr, exists := c.Get("user_id")
	if exists {
		if uid, err := uuid.Parse(userIDStr.(string)); err == nil {
			userIDPtr = &uid
		}
	}

	resp, err := h.service.GetReactions(c.Request.Context(), userIDPtr, refID, refType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
