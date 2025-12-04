package handler

import (
	"net/http"

	"anoa.com/telkomalumiforum/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminService service.AdminService
}

func NewAdminHandler(adminService service.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var input service.CreateUserInput
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

	res, err := h.adminService.CreateUser(c.Request.Context(), input, avatar)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}
