package repository

import (
	"context"
	"errors"

	"anoa.com/telkomalumiforum/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrAlreadyOwned = errors.New("kosmetik ini sudah kamu miliki")

type CosmeticRepository interface {
	GetPublishedCatalog(ctx context.Context) ([]entity.Cosmetic, error)
	// GetAll returns every cosmetic regardless of status (draft/published/
	// retired) — admin-only, unlike GetPublishedCatalog.
	GetAll(ctx context.Context) ([]entity.Cosmetic, error)
	GetByID(ctx context.Context, id uint) (*entity.Cosmetic, error)
	Create(ctx context.Context, cosmetic *entity.Cosmetic) error
	Update(ctx context.Context, cosmetic *entity.Cosmetic) error

	GetOwnership(ctx context.Context, userID uuid.UUID, cosmeticID uint) (*entity.CosmeticOwnership, error)
	// CreateOwnershipInTx inserts an ownership row within the purchase
	// transaction; returns ErrAlreadyOwned if the (user, cosmetic) pair
	// already exists instead of a raw unique-constraint error.
	CreateOwnershipInTx(tx *gorm.DB, userID uuid.UUID, cosmeticID uint) error
	GetOwnershipsForUser(ctx context.Context, userID uuid.UUID) ([]entity.CosmeticOwnership, error)

	GetUserEquip(ctx context.Context, userID uuid.UUID) (*entity.UserEquip, error)
	GetUserEquipsByUsernames(ctx context.Context, usernames []string) (map[string]*entity.UserEquip, error)
	UpsertEquip(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error

	WithTx(tx *gorm.DB) CosmeticRepository
	DB() *gorm.DB
}

type cosmeticRepository struct {
	db *gorm.DB
}

func NewCosmeticRepository(db *gorm.DB) CosmeticRepository {
	return &cosmeticRepository{db: db}
}

func (r *cosmeticRepository) DB() *gorm.DB {
	return r.db
}

func (r *cosmeticRepository) WithTx(tx *gorm.DB) CosmeticRepository {
	return &cosmeticRepository{db: tx}
}

func (r *cosmeticRepository) GetPublishedCatalog(ctx context.Context) ([]entity.Cosmetic, error) {
	var cosmetics []entity.Cosmetic
	err := r.db.WithContext(ctx).Where("status = ?", "published").Find(&cosmetics).Error
	return cosmetics, err
}

func (r *cosmeticRepository) GetAll(ctx context.Context) ([]entity.Cosmetic, error) {
	var cosmetics []entity.Cosmetic
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&cosmetics).Error
	return cosmetics, err
}

func (r *cosmeticRepository) GetByID(ctx context.Context, id uint) (*entity.Cosmetic, error) {
	var cosmetic entity.Cosmetic
	if err := r.db.WithContext(ctx).First(&cosmetic, id).Error; err != nil {
		return nil, err
	}
	return &cosmetic, nil
}

func (r *cosmeticRepository) Create(ctx context.Context, cosmetic *entity.Cosmetic) error {
	return r.db.WithContext(ctx).Create(cosmetic).Error
}

func (r *cosmeticRepository) Update(ctx context.Context, cosmetic *entity.Cosmetic) error {
	return r.db.WithContext(ctx).Save(cosmetic).Error
}

func (r *cosmeticRepository) GetOwnership(ctx context.Context, userID uuid.UUID, cosmeticID uint) (*entity.CosmeticOwnership, error) {
	var ownership entity.CosmeticOwnership
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND cosmetic_id = ?", userID, cosmeticID).
		First(&ownership).Error
	if err != nil {
		return nil, err
	}
	return &ownership, nil
}

func (r *cosmeticRepository) CreateOwnershipInTx(tx *gorm.DB, userID uuid.UUID, cosmeticID uint) error {
	var count int64
	if err := tx.Model(&entity.CosmeticOwnership{}).
		Where("user_id = ? AND cosmetic_id = ?", userID, cosmeticID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrAlreadyOwned
	}

	return tx.Create(&entity.CosmeticOwnership{UserID: userID, CosmeticID: cosmeticID}).Error
}

func (r *cosmeticRepository) GetOwnershipsForUser(ctx context.Context, userID uuid.UUID) ([]entity.CosmeticOwnership, error) {
	var ownerships []entity.CosmeticOwnership
	err := r.db.WithContext(ctx).
		Preload("Cosmetic").
		Where("user_id = ?", userID).
		Find(&ownerships).Error
	return ownerships, err
}

func (r *cosmeticRepository) GetUserEquip(ctx context.Context, userID uuid.UUID) (*entity.UserEquip, error) {
	var equip entity.UserEquip
	err := r.db.WithContext(ctx).
		Preload("AvatarBorder").Preload("ThreadBg").Preload("ProfileBg").
		Where("user_id = ?", userID).
		First(&equip).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &entity.UserEquip{UserID: userID}, nil
		}
		return nil, err
	}
	return &equip, nil
}

func (r *cosmeticRepository) GetUserEquipsByUsernames(ctx context.Context, usernames []string) (map[string]*entity.UserEquip, error) {
	var users []entity.User
	if err := r.db.WithContext(ctx).
		Select("id", "username").
		Where("username IN ?", usernames).
		Find(&users).Error; err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return map[string]*entity.UserEquip{}, nil
	}

	userIDs := make([]uuid.UUID, len(users))
	usernameByID := make(map[uuid.UUID]string, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
		usernameByID[u.ID] = u.Username
	}

	var equips []entity.UserEquip
	if err := r.db.WithContext(ctx).
		Preload("AvatarBorder").Preload("ThreadBg").Preload("ProfileBg").
		Where("user_id IN ?", userIDs).
		Find(&equips).Error; err != nil {
		return nil, err
	}

	result := make(map[string]*entity.UserEquip, len(equips))
	for i := range equips {
		result[usernameByID[equips[i].UserID]] = &equips[i]
	}
	return result, nil
}

func (r *cosmeticRepository) UpsertEquip(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error {
	base := entity.UserEquip{UserID: userID}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(updates),
	}).Create(&base).Error
}
