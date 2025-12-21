package repository

import (
	"context"

	"anoa.com/telkomalumiforum/internal/model"
	"gorm.io/gorm"
)

type MenfessRepository interface {
	Create(ctx context.Context, menfess *model.Menfess) error
	FindAll(ctx context.Context, offset, limit int) ([]*model.Menfess, int64, error)
}

type menfessRepository struct {
	db *gorm.DB
}

func NewMenfessRepository(db *gorm.DB) MenfessRepository {
	return &menfessRepository{db: db}
}

func (r *menfessRepository) Create(ctx context.Context, menfess *model.Menfess) error {
	return r.db.WithContext(ctx).Create(menfess).Error
}

func (r *menfessRepository) FindAll(ctx context.Context, offset, limit int) ([]*model.Menfess, int64, error) {
	var menfesses []*model.Menfess
	var total int64

	if err := r.db.Model(&model.Menfess{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&menfesses).Error; err != nil {
		return nil, 0, err
	}

	return menfesses, total, nil
}
