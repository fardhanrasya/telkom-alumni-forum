package repository

import (
	"context"

	"anoa.com/telkomalumiforum/internal/model"
	"gorm.io/gorm"
)

type ThreadRepository interface {
	Create(ctx context.Context, thread *model.Thread) error
	FindBySlug(ctx context.Context, slug string) (*model.Thread, error)
}

type threadRepository struct {
	db *gorm.DB
}

func NewThreadRepository(db *gorm.DB) ThreadRepository {
	return &threadRepository{db: db}
}

func (r *threadRepository) Create(ctx context.Context, thread *model.Thread) error {
	return r.db.WithContext(ctx).Create(thread).Error
}

func (r *threadRepository) FindBySlug(ctx context.Context, slug string) (*model.Thread, error) {
	var thread model.Thread
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&thread).Error; err != nil {
		return nil, err
	}
	return &thread, nil
}
