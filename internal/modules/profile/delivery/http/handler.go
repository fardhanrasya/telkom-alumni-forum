package handler

import (
	"net/http"

	profileDto "anoa.com/telkomalumiforum/internal/modules/profile/dto"
	profile "anoa.com/telkomalumiforum/internal/modules/profile/service"
	commonDto "anoa.com/telkomalumiforum/pkg/dto"
	"anoa.com/telkomalumiforum/pkg/validator"
	"github.com/gin-gonic/gin"
)

type ProfileHandler struct {
	profileService profile.ProfileService
}

func NewProfileHandler(profileService profile.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

func (h *ProfileHandler) GetProfileByUsername(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username diperlukan"})
		return
	}

	profile, err := h.profileService.GetProfileByUsername(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) GetCurrentProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user tidak terautentikasi"})
		return
	}

	profile, err := h.profileService.GetCurrentProfile(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user tidak terautentikasi"})
		return
	}

	var input profileDto.UpdateProfileInput
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.FormatValidationError(err)})
		return
	}

	var avatar *commonDto.AvatarFile
	if fileHeader, err := c.FormFile("avatar"); err == nil && fileHeader != nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gagal memuat avatar"})
			return
		}
		defer file.Close()

		avatar = &commonDto.AvatarFile{
			Reader:   file,
			FileName: fileHeader.Filename,
		}
	}

	res, err := h.profileService.UpdateProfile(c.Request.Context(), userID.(string), input, avatar)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
