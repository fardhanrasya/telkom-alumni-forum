package handler

import (
	"net/http"
	"strconv"

	"anoa.com/telkomalumiforum/internal/dto"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MenfessHandler struct {
	service  service.MenfessService
	userRepo repository.UserRepository
}

func NewMenfessHandler(service service.MenfessService, userRepo repository.UserRepository) *MenfessHandler {
	return &MenfessHandler{service: service, userRepo: userRepo}
}

func (h *MenfessHandler) CreateMenfess(c *gin.Context) {
	var req dto.CreateMenfessRequest
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

	user, err := h.userRepo.FindByID(c.Request.Context(), userID.String())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	if user.Role.Name == "guru" {
		c.JSON(http.StatusForbidden, gin.H{"error": "guru cannot post menfess"})
		return
	}

	if err := h.service.CreateMenfess(c.Request.Context(), userID, req.Content); err != nil {
		if err.Error() == "menfess quota exceeded (max 2 per day)" {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "menfess created successfully"})
}

func (h *MenfessHandler) GetMenfesses(c *gin.Context) {
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

	// TODO: Performance Improvement
	// Saat ini kita query DB untuk cek role. Untuk skala besar,
	// sebaiknya role disimpan di JWT Claims context untuk menghindari DB call
	user, err := h.userRepo.FindByID(c.Request.Context(), userID.String())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	if user.Role.Name == "guru" {
		c.JSON(http.StatusForbidden, gin.H{"error": "guru cannot view menfess"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	menfesses, total, err := h.service.GetMenfesses(c.Request.Context(), &userID, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  menfesses,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

