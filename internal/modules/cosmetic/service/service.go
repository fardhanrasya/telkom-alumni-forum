package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"anoa.com/telkomalumiforum/internal/entity"
	cosmeticDto "anoa.com/telkomalumiforum/internal/modules/cosmetic/dto"
	cosmeticRepo "anoa.com/telkomalumiforum/internal/modules/cosmetic/repository"
	leaderboardRepo "anoa.com/telkomalumiforum/internal/modules/leaderboard/repository"
	leaderboardService "anoa.com/telkomalumiforum/internal/modules/leaderboard/service"
	userRepo "anoa.com/telkomalumiforum/internal/modules/user/repository"
	walletRepo "anoa.com/telkomalumiforum/internal/modules/wallet/repository"
	"anoa.com/telkomalumiforum/pkg/apperror"
	commonDto "anoa.com/telkomalumiforum/pkg/dto"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type InsufficientBalanceError struct {
	Balance int
	Price   int
}

func (e *InsufficientBalanceError) Error() string { return "saldo TC tidak cukup" }

type RankTooLowError struct {
	CurrentRank string
	MinRank     string
}

func (e *RankTooLowError) Error() string { return "rank belum cukup" }

var ErrInvalidPayload = errors.New("payload aset tidak valid untuk render_type ini")

type CosmeticService interface {
	GetCatalog(ctx context.Context) ([]cosmeticDto.CosmeticResponse, error)
	GetUserCosmeticsByUsername(ctx context.Context, username string) (*cosmeticDto.EquipResponse, error)
	BatchGetCosmetics(ctx context.Context, usernames []string) (map[string]*cosmeticDto.EquipResponse, error)
	Purchase(ctx context.Context, userID uuid.UUID, cosmeticID uint) error
	GetInventory(ctx context.Context, userID uuid.UUID) ([]cosmeticDto.InventoryItemResponse, error)
	Equip(ctx context.Context, userID uuid.UUID, slot string, cosmeticID *uint) error

	CreateCosmetic(ctx context.Context, input cosmeticDto.AdminCosmeticInput, animated, static *commonDto.AvatarFile) (*cosmeticDto.CosmeticResponse, error)
	UpdateCosmetic(ctx context.Context, id uint, input cosmeticDto.AdminCosmeticInput, animated, static *commonDto.AvatarFile) (*cosmeticDto.CosmeticResponse, error)
}

type cosmeticService struct {
	repo            cosmeticRepo.CosmeticRepository
	walletRepo      walletRepo.WalletRepository
	leaderboardRepo leaderboardRepo.LeaderboardRepository
	userRepo        userRepo.UserRepository
	imageStorage    storage.ImageStorage
}

func NewCosmeticService(
	repo cosmeticRepo.CosmeticRepository,
	walletRepo walletRepo.WalletRepository,
	leaderboardRepo leaderboardRepo.LeaderboardRepository,
	userRepo userRepo.UserRepository,
	imageStorage storage.ImageStorage,
) CosmeticService {
	return &cosmeticService{
		repo:            repo,
		walletRepo:      walletRepo,
		leaderboardRepo: leaderboardRepo,
		userRepo:        userRepo,
		imageStorage:    imageStorage,
	}
}

func toCosmeticResponse(c *entity.Cosmetic) cosmeticDto.CosmeticResponse {
	resp := cosmeticDto.CosmeticResponse{
		ID:         c.ID,
		Slot:       c.Slot,
		SubType:    c.SubType,
		RenderType: c.RenderType,
		Name:       c.Name,
		Price:      c.Price,
		MinRank:    c.MinRank,
		Status:     c.Status,
	}
	switch c.RenderType {
	case "css":
		var p cosmeticDto.CSSPayload
		if err := json.Unmarshal(c.Payload, &p); err == nil {
			resp.Payload = p
		}
	case "image":
		var p cosmeticDto.ImagePayload
		if err := json.Unmarshal(c.Payload, &p); err == nil {
			resp.Payload = p
		}
	}
	return resp
}

func toEquipResponse(equip *entity.UserEquip) *cosmeticDto.EquipResponse {
	resp := &cosmeticDto.EquipResponse{}
	if equip.AvatarBorder != nil {
		r := toCosmeticResponse(equip.AvatarBorder)
		resp.AvatarBorder = &r
	}
	if equip.ThreadBg != nil {
		r := toCosmeticResponse(equip.ThreadBg)
		resp.ThreadBg = &r
	}
	if equip.ProfileBg != nil {
		r := toCosmeticResponse(equip.ProfileBg)
		resp.ProfileBg = &r
	}
	return resp
}

func (s *cosmeticService) GetCatalog(ctx context.Context) ([]cosmeticDto.CosmeticResponse, error) {
	cosmetics, err := s.repo.GetPublishedCatalog(ctx)
	if err != nil {
		return nil, err
	}
	responses := make([]cosmeticDto.CosmeticResponse, 0, len(cosmetics))
	for i := range cosmetics {
		responses = append(responses, toCosmeticResponse(&cosmetics[i]))
	}
	return responses, nil
}

func (s *cosmeticService) GetUserCosmeticsByUsername(ctx context.Context, username string) (*cosmeticDto.EquipResponse, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, apperror.ErrNotFound
	}

	equip, err := s.repo.GetUserEquip(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return toEquipResponse(equip), nil
}

func (s *cosmeticService) BatchGetCosmetics(ctx context.Context, usernames []string) (map[string]*cosmeticDto.EquipResponse, error) {
	equips, err := s.repo.GetUserEquipsByUsernames(ctx, usernames)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*cosmeticDto.EquipResponse, len(equips))
	for username, equip := range equips {
		result[username] = toEquipResponse(equip)
	}
	return result, nil
}

func (s *cosmeticService) currentRank(userID uuid.UUID) (string, error) {
	stats, err := s.leaderboardRepo.GetUserStatsByUserID(userID)
	if err != nil {
		return "", err
	}
	points := 0
	if stats != nil {
		points = stats.TotalScoreAllTime
	}
	return leaderboardService.GetGamificationStatus(points).RankName, nil
}

func (s *cosmeticService) Purchase(ctx context.Context, userID uuid.UUID, cosmeticID uint) error {
	cosmetic, err := s.repo.GetByID(ctx, cosmeticID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrNotFound
		}
		return err
	}
	if cosmetic.Status != "published" {
		return apperror.ErrNotFound
	}

	if _, err := s.repo.GetOwnership(ctx, userID, cosmeticID); err == nil {
		return cosmeticRepo.ErrAlreadyOwned
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	rank, err := s.currentRank(userID)
	if err != nil {
		return err
	}
	if !meetsMinRank(rank, cosmetic.MinRank) {
		return &RankTooLowError{CurrentRank: rank, MinRank: cosmetic.MinRank}
	}

	sourceRefID := fmt.Sprintf("%s:%d", userID, cosmeticID)
	err = s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := s.walletRepo.WithTx(tx).ApplyLedgerEntry(tx, userID, -cosmetic.Price, "purchase", sourceRefID); err != nil {
			return err
		}
		return s.repo.WithTx(tx).CreateOwnershipInTx(tx, userID, cosmeticID)
	})
	if err != nil {
		if errors.Is(err, walletRepo.ErrInsufficientBalance) {
			wallet, wErr := s.walletRepo.GetOrCreateWallet(ctx, userID)
			balance := 0
			if wErr == nil {
				balance = wallet.Balance
			}
			return &InsufficientBalanceError{Balance: balance, Price: cosmetic.Price}
		}
		return err
	}

	return nil
}

func (s *cosmeticService) GetInventory(ctx context.Context, userID uuid.UUID) ([]cosmeticDto.InventoryItemResponse, error) {
	ownerships, err := s.repo.GetOwnershipsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]cosmeticDto.InventoryItemResponse, 0, len(ownerships))
	for _, o := range ownerships {
		if o.Cosmetic == nil {
			continue
		}
		items = append(items, cosmeticDto.InventoryItemResponse{
			Cosmetic:    toCosmeticResponse(o.Cosmetic),
			PurchasedAt: o.PurchasedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return items, nil
}

func (s *cosmeticService) Equip(ctx context.Context, userID uuid.UUID, slot string, cosmeticID *uint) error {
	column, ok := map[string]string{
		"avatar_border": "avatar_border_id",
		"thread_bg":     "thread_bg_id",
		"profile_bg":    "profile_bg_id",
	}[slot]
	if !ok {
		return apperror.New(400, "slot tidak valid", apperror.ErrInvalidInput)
	}

	if cosmeticID != nil {
		cosmetic, err := s.repo.GetByID(ctx, *cosmeticID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.ErrNotFound
			}
			return err
		}
		if cosmetic.Slot != slot {
			return apperror.New(400, "kosmetik tidak cocok dengan slot ini", apperror.ErrInvalidInput)
		}
		if _, err := s.repo.GetOwnership(ctx, userID, *cosmeticID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.New(403, "kamu belum memiliki kosmetik ini", apperror.ErrForbidden)
			}
			return err
		}
	}

	return s.repo.UpsertEquip(ctx, userID, map[string]interface{}{column: cosmeticID})
}

func buildPayload(input cosmeticDto.AdminCosmeticInput, animatedURL, staticURL string) (datatypes.JSON, error) {
	switch input.RenderType {
	case "css":
		if input.PresetKey == "" {
			return nil, ErrInvalidPayload
		}
		return json.Marshal(cosmeticDto.CSSPayload{PresetKey: input.PresetKey})
	case "image":
		if animatedURL == "" || staticURL == "" {
			return nil, ErrInvalidPayload
		}
		return json.Marshal(cosmeticDto.ImagePayload{AnimatedURL: animatedURL, StaticURL: staticURL})
	default:
		return nil, ErrInvalidPayload
	}
}

func (s *cosmeticService) uploadIfPresent(ctx context.Context, file *commonDto.AvatarFile, folder string) (string, error) {
	if file == nil {
		return "", nil
	}
	reader, ok := file.Reader.(io.Reader)
	if !ok {
		return "", ErrInvalidPayload
	}
	return s.imageStorage.UploadImage(ctx, reader, folder, file.FileName)
}

func (s *cosmeticService) CreateCosmetic(ctx context.Context, input cosmeticDto.AdminCosmeticInput, animated, static *commonDto.AvatarFile) (*cosmeticDto.CosmeticResponse, error) {
	if input.RenderType == "css" && !(input.Slot == "thread_bg" || (input.Slot == "avatar_border" && input.SubType == "ring")) {
		return nil, ErrInvalidPayload
	}
	if input.RenderType == "image" && !(input.Slot == "profile_bg" || (input.Slot == "avatar_border" && input.SubType == "decoration")) {
		return nil, ErrInvalidPayload
	}

	animatedURL, err := s.uploadIfPresent(ctx, animated, "cosmetics")
	if err != nil {
		return nil, err
	}
	staticURL, err := s.uploadIfPresent(ctx, static, "cosmetics")
	if err != nil {
		return nil, err
	}

	payload, err := buildPayload(input, animatedURL, staticURL)
	if err != nil {
		return nil, err
	}

	status := input.Status
	if status == "" {
		status = "draft"
	}

	cosmetic := &entity.Cosmetic{
		Slot:       input.Slot,
		SubType:    input.SubType,
		RenderType: input.RenderType,
		Name:       input.Name,
		Payload:    payload,
		Price:      input.Price,
		MinRank:    input.MinRank,
		Status:     status,
	}

	if err := s.repo.Create(ctx, cosmetic); err != nil {
		return nil, err
	}

	resp := toCosmeticResponse(cosmetic)
	return &resp, nil
}

func (s *cosmeticService) UpdateCosmetic(ctx context.Context, id uint, input cosmeticDto.AdminCosmeticInput, animated, static *commonDto.AvatarFile) (*cosmeticDto.CosmeticResponse, error) {
	cosmetic, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, err
	}

	animatedURL, err := s.uploadIfPresent(ctx, animated, "cosmetics")
	if err != nil {
		return nil, err
	}
	staticURL, err := s.uploadIfPresent(ctx, static, "cosmetics")
	if err != nil {
		return nil, err
	}

	if animatedURL == "" || staticURL == "" {
		// Keep existing image URLs when no replacement file was uploaded.
		var existing cosmeticDto.ImagePayload
		if json.Unmarshal(cosmetic.Payload, &existing) == nil {
			if animatedURL == "" {
				animatedURL = existing.AnimatedURL
			}
			if staticURL == "" {
				staticURL = existing.StaticURL
			}
		}
	}

	payload, err := buildPayload(input, animatedURL, staticURL)
	if err != nil {
		return nil, err
	}

	cosmetic.Slot = input.Slot
	cosmetic.SubType = input.SubType
	cosmetic.RenderType = input.RenderType
	cosmetic.Name = input.Name
	cosmetic.Payload = payload
	cosmetic.Price = input.Price
	cosmetic.MinRank = input.MinRank
	if input.Status != "" {
		cosmetic.Status = input.Status
	}

	if err := s.repo.Update(ctx, cosmetic); err != nil {
		return nil, err
	}

	resp := toCosmeticResponse(cosmetic)
	return &resp, nil
}
