package repository

import (
	"context"
	"errors"

	"anoa.com/telkomalumiforum/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MissionRepository interface {
	GetActiveDefinitionsByActionType(ctx context.Context, actionType string) ([]entity.MissionDefinition, error)
	GetActiveDefinitions(ctx context.Context) ([]entity.MissionDefinition, error)
	GetDefinitionByID(ctx context.Context, id uint) (*entity.MissionDefinition, error)
	GetProgressForUser(ctx context.Context, userID uuid.UUID, periods []string) ([]entity.MissionProgress, error)
	// IncrementProgress upserts a progress row, adding delta and capping at
	// cap (the mission's target) so an over-fire never over-counts.
	IncrementProgress(ctx context.Context, userID uuid.UUID, missionID uint, period string, delta, cap int) error
	// SetProgress upserts a progress row to an absolute value (clamped to
	// [0, cap]), unlike IncrementProgress's additive accumulation. Needed
	// for streak-style missions where progress can legitimately go down
	// (a broken streak), which an additive counter could never express.
	SetProgress(ctx context.Context, userID uuid.UUID, missionID uint, period string, value, cap int) error
	// GetProgressForUpdate locks the progress row (creating it at 0 if
	// missing) for the claim transaction.
	GetProgressForUpdate(tx *gorm.DB, userID uuid.UUID, missionID uint, period string) (*entity.MissionProgress, error)
	MarkClaimed(tx *gorm.DB, progressID uint) error
	WithTx(tx *gorm.DB) MissionRepository
	DB() *gorm.DB
}

type missionRepository struct {
	db *gorm.DB
}

func NewMissionRepository(db *gorm.DB) MissionRepository {
	return &missionRepository{db: db}
}

func (r *missionRepository) DB() *gorm.DB {
	return r.db
}

func (r *missionRepository) WithTx(tx *gorm.DB) MissionRepository {
	return &missionRepository{db: tx}
}

func (r *missionRepository) GetActiveDefinitionsByActionType(ctx context.Context, actionType string) ([]entity.MissionDefinition, error) {
	var defs []entity.MissionDefinition
	err := r.db.WithContext(ctx).
		Where("action_type = ? AND is_active = ?", actionType, true).
		Find(&defs).Error
	return defs, err
}

func (r *missionRepository) GetActiveDefinitions(ctx context.Context) ([]entity.MissionDefinition, error) {
	var defs []entity.MissionDefinition
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&defs).Error
	return defs, err
}

func (r *missionRepository) GetDefinitionByID(ctx context.Context, id uint) (*entity.MissionDefinition, error) {
	var def entity.MissionDefinition
	if err := r.db.WithContext(ctx).First(&def, id).Error; err != nil {
		return nil, err
	}
	return &def, nil
}

func (r *missionRepository) GetProgressForUser(ctx context.Context, userID uuid.UUID, periods []string) ([]entity.MissionProgress, error) {
	var progress []entity.MissionProgress
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND period IN ?", userID, periods).
		Find(&progress).Error
	return progress, err
}

func (r *missionRepository) IncrementProgress(ctx context.Context, userID uuid.UUID, missionID uint, period string, delta, cap int) error {
	initial := delta
	if initial > cap {
		initial = cap
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "mission_id"}, {Name: "period"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"progress": gorm.Expr("LEAST(?, progress + ?)", cap, delta),
		}),
	}).Create(&entity.MissionProgress{
		UserID:    userID,
		MissionID: missionID,
		Period:    period,
		Progress:  initial,
	}).Error
}

func (r *missionRepository) SetProgress(ctx context.Context, userID uuid.UUID, missionID uint, period string, value, cap int) error {
	if value > cap {
		value = cap
	}
	if value < 0 {
		value = 0
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "mission_id"}, {Name: "period"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"progress": value,
		}),
	}).Create(&entity.MissionProgress{
		UserID:    userID,
		MissionID: missionID,
		Period:    period,
		Progress:  value,
	}).Error
}

func (r *missionRepository) GetProgressForUpdate(tx *gorm.DB, userID uuid.UUID, missionID uint, period string) (*entity.MissionProgress, error) {
	var progress entity.MissionProgress
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND mission_id = ? AND period = ?", userID, missionID, period).
		First(&progress).Error
	if err == nil {
		return &progress, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	progress = entity.MissionProgress{UserID: userID, MissionID: missionID, Period: period, Progress: 0}
	if err := tx.Create(&progress).Error; err != nil {
		return nil, err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND mission_id = ? AND period = ?", userID, missionID, period).
		First(&progress).Error; err != nil {
		return nil, err
	}
	return &progress, nil
}

func (r *missionRepository) MarkClaimed(tx *gorm.DB, progressID uint) error {
	return tx.Model(&entity.MissionProgress{}).
		Where("id = ?", progressID).
		Update("claimed_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}
