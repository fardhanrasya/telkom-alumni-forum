package handler

import (
	"net/http"

	"anoa.com/telkomalumiforum/internal/service"
	"github.com/gin-gonic/gin"
)

type ProfileHandler struct {
	profileService service.ProfileService
}

func NewProfileHandler(profileService service.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user tidak terautentikasi"})
		return
	}

	var input service.UpdateProfileInput
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
		return
	}

	var avatar *service.AvatarFile
	if fileHeader, err := c.FormFile("avatar"); err == nil && fileHeader != nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gagal memuat avatar"})
			return
		}
		defer file.Close()

		avatar = &service.AvatarFile{
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
