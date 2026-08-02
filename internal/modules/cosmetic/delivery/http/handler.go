package http

import (
	"errors"
	"io"
	"net/http"

	cosmeticDto "anoa.com/telkomalumiforum/internal/modules/cosmetic/dto"
	cosmeticRepo "anoa.com/telkomalumiforum/internal/modules/cosmetic/repository"
	cosmeticService "anoa.com/telkomalumiforum/internal/modules/cosmetic/service"
	commonDto "anoa.com/telkomalumiforum/pkg/dto"
	"anoa.com/telkomalumiforum/pkg/response"
	"anoa.com/telkomalumiforum/pkg/validator"
	"github.com/gin-gonic/gin"
)

type CosmeticHandler struct {
	service cosmeticService.CosmeticService
}

func NewCosmeticHandler(service cosmeticService.CosmeticService) *CosmeticHandler {
	return &CosmeticHandler{service: service}
}

func (h *CosmeticHandler) GetCatalog(c *gin.Context) {
	catalog, err := h.service.GetCatalog(c.Request.Context())
	if err != nil {
		response.ResponseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": catalog})
}

func (h *CosmeticHandler) ListCosmeticsAdmin(c *gin.Context) {
	catalog, err := h.service.GetAllForAdmin(c.Request.Context())
	if err != nil {
		response.ResponseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": catalog})
}

func (h *CosmeticHandler) GetUserCosmetics(c *gin.Context) {
	username := c.Param("username")
	equip, err := h.service.GetUserCosmeticsByUsername(c.Request.Context(), username)
	if err != nil {
		response.ResponseError(c, err)
		return
	}
	c.JSON(http.StatusOK, equip)
}

func (h *CosmeticHandler) BatchGetCosmetics(c *gin.Context) {
	var req cosmeticDto.BatchCosmeticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.FormatValidationError(err)})
		return
	}

	result, err := h.service.BatchGetCosmetics(c.Request.Context(), req.Usernames)
	if err != nil {
		response.ResponseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *CosmeticHandler) Purchase(c *gin.Context) {
	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id kosmetik tidak valid"})
		return
	}

	err = h.service.Purchase(c.Request.Context(), userID, uri.ID)
	if err != nil {
		var insufficientErr *cosmeticService.InsufficientBalanceError
		var rankErr *cosmeticService.RankTooLowError

		switch {
		case errors.As(err, &insufficientErr):
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "saldo_tidak_cukup",
				"message": "Saldo TC tidak cukup",
				"balance": insufficientErr.Balance,
				"price":   insufficientErr.Price,
			})
		case errors.As(err, &rankErr):
			c.JSON(http.StatusBadRequest, gin.H{
				"error":        "rank_tidak_cukup",
				"message":      "Rank minimal " + rankErr.MinRank,
				"current_rank": rankErr.CurrentRank,
				"min_rank":     rankErr.MinRank,
			})
		case errors.Is(err, cosmeticRepo.ErrAlreadyOwned):
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah_dimiliki", "message": err.Error()})
		default:
			response.ResponseError(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pembelian berhasil"})
}

func (h *CosmeticHandler) GetInventory(c *gin.Context) {
	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	items, err := h.service.GetInventory(c.Request.Context(), userID)
	if err != nil {
		response.ResponseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *CosmeticHandler) Equip(c *gin.Context) {
	h.setEquip(c, true)
}

func (h *CosmeticHandler) Unequip(c *gin.Context) {
	h.setEquip(c, false)
}

func (h *CosmeticHandler) setEquip(c *gin.Context, requireCosmeticID bool) {
	userID, err := response.GetUserID(c)
	if err != nil {
		response.ResponseError(c, err)
		return
	}

	var req cosmeticDto.EquipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.FormatValidationError(err)})
		return
	}

	cosmeticID := req.CosmeticID
	if !requireCosmeticID {
		cosmeticID = nil
	} else if cosmeticID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cosmetic_id wajib diisi"})
		return
	}

	if err := h.service.Equip(c.Request.Context(), userID, req.Slot, cosmeticID); err != nil {
		response.ResponseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "berhasil diperbarui"})
}

func (h *CosmeticHandler) CreateCosmetic(c *gin.Context) {
	var input cosmeticDto.AdminCosmeticInput
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.FormatValidationError(err)})
		return
	}

	animated, static, closeFiles, err := openCosmeticFiles(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gagal memuat file aset"})
		return
	}
	defer closeFiles()

	res, err := h.service.CreateCosmetic(c.Request.Context(), input, animated, static)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *CosmeticHandler) UpdateCosmetic(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id kosmetik tidak valid"})
		return
	}

	var input cosmeticDto.AdminCosmeticInput
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.FormatValidationError(err)})
		return
	}

	animated, static, closeFiles, err := openCosmeticFiles(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gagal memuat file aset"})
		return
	}
	defer closeFiles()

	res, err := h.service.UpdateCosmetic(c.Request.Context(), uri.ID, input, animated, static)
	if err != nil {
		response.ResponseError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func openCosmeticFiles(c *gin.Context) (animated, static *commonDto.AvatarFile, closeFn func(), err error) {
	var closers []io.Closer
	closeFn = func() {
		for _, cl := range closers {
			cl.Close()
		}
	}

	if fileHeader, ferr := c.FormFile("animated"); ferr == nil && fileHeader != nil {
		f, oerr := fileHeader.Open()
		if oerr != nil {
			return nil, nil, closeFn, oerr
		}
		closers = append(closers, f)
		animated = &commonDto.AvatarFile{Reader: f, FileName: fileHeader.Filename}
	}

	if fileHeader, ferr := c.FormFile("static"); ferr == nil && fileHeader != nil {
		f, oerr := fileHeader.Open()
		if oerr != nil {
			return nil, nil, closeFn, oerr
		}
		closers = append(closers, f)
		static = &commonDto.AvatarFile{Reader: f, FileName: fileHeader.Filename}
	}

	return animated, static, closeFn, nil
}
