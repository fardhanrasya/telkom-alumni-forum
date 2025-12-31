package handler

import (
	"net/http"

	"anoa.com/telkomalumiforum/internal/modules/admin/dto"
	adminService "anoa.com/telkomalumiforum/internal/modules/admin/service"
	commonDto "anoa.com/telkomalumiforum/pkg/dto"
	"anoa.com/telkomalumiforum/pkg/validator"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminService adminService.AdminService
}

func NewAdminHandler(adminService adminService.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var input dto.CreateUserInput
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

	res, err := h.adminService.CreateUser(c.Request.Context(), input, avatar)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	res, err := h.adminService.GetAllUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.adminService.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var input dto.UpdateAdminUserInput
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

	res, err := h.adminService.UpdateUser(c.Request.Context(), id, input, avatar)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
