package repository

import (
	"errors"

	"anoa.com/telkomalumiforum/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ActivityRepository interface {
	// GetStreakForUpdate locks the user's streak row (creating it if
	// missing) for the heartbeat transaction.
	GetStreakForUpdate(tx *gorm.DB, userID uuid.UUID) (*entity.LoginStreak, error)
	SaveStreak(tx *gorm.DB, streak *entity.LoginStreak) error
	DB() *gorm.DB
}

type activityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &activityRepository{db: db}
}

func (r *activityRepository) DB() *gorm.DB {
	return r.db
}

func (r *activityRepository) GetStreakForUpdate(tx *gorm.DB, userID uuid.UUID) (*entity.LoginStreak, error) {
	var streak entity.LoginStreak
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).First(&streak).Error
	if err == nil {
		return &streak, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	streak = entity.LoginStreak{UserID: userID, LastActiveDate: "", CurrentStreak: 0}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&streak).Error; err != nil {
		return nil, err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).First(&streak).Error; err != nil {
		return nil, err
	}
	return &streak, nil
}

func (r *activityRepository) SaveStreak(tx *gorm.DB, streak *entity.LoginStreak) error {
	return tx.Save(streak).Error
}
