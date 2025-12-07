package repository

import (
	"context"

	"anoa.com/telkomalumiforum/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ThreadRepository interface {
	Create(ctx context.Context, thread *model.Thread) error
	FindBySlug(ctx context.Context, slug string) (*model.Thread, error)
	FindAll(ctx context.Context, categoryID *uuid.UUID, search string, audiences []string, sortBy string, offset, limit int) ([]*model.Thread, int64, error)
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
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("User").
		Preload("User.Profile").
		Preload("Attachments").
		Where("slug = ?", slug).
		First(&thread).Error; err != nil {
		return nil, err
	}
	return &thread, nil
}

func (r *threadRepository) FindAll(ctx context.Context, categoryID *uuid.UUID, search string, audiences []string, sortBy string, offset, limit int) ([]*model.Thread, int64, error) {
	var threads []*model.Thread
	var total int64
	
	query := r.db.WithContext(ctx).
		Preload("Category").
		Preload("User").
		Preload("User.Profile").
		Preload("Attachments")

	if categoryID != nil {
		query = query.Where("category_id = ?", categoryID)
	}

	if search != "" {
		query = query.Where("title ILIKE ? OR content ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if len(audiences) > 0 {
		query = query.Where("audience IN ?", audiences)
	}

	if err := query.Model(&model.Thread{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if sortBy == "popular" {
		query = query.Order("views DESC").Order("created_at DESC")
	} else {
		query = query.Order("created_at DESC")
	}

	if err := query.Offset(offset).Limit(limit).Find(&threads).Error; err != nil {
		return nil, 0, err
	}

	return threads, total, nil
}
