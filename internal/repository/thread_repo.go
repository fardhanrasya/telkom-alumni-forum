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
	FindByID(ctx context.Context, id uuid.UUID) (*model.Thread, error)
	FindAll(ctx context.Context, categoryID *uuid.UUID, search string, audiences []string, sortBy string, offset, limit int) ([]*model.Thread, int64, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, audiences []string, offset, limit int) ([]*model.Thread, int64, error)
	GetTrending(ctx context.Context, limit int) ([]*model.Thread, error)
	Update(ctx context.Context, thread *model.Thread) error
	Delete(ctx context.Context, id uuid.UUID) error
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

func (r *threadRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Thread, error) {
	var thread model.Thread
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("User").
		Preload("User.Profile").
		Preload("Attachments").
		Where("id = ?", id).
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

func (r *threadRepository) FindByUserID(ctx context.Context, userID uuid.UUID, audiences []string, offset, limit int) ([]*model.Thread, int64, error) {
	var threads []*model.Thread
	var total int64

	query := r.db.WithContext(ctx).
		Preload("Category").
		Preload("User").
		Preload("User.Profile").
		Preload("Attachments").
		Where("user_id = ?", userID)

	if len(audiences) > 0 {
		query = query.Where("audience IN ?", audiences)
	}

	if err := query.Model(&model.Thread{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&threads).Error; err != nil {
		return nil, 0, err
	}

	return threads, total, nil
}

func (r *threadRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Thread{}, id).Error
}

func (r *threadRepository) Update(ctx context.Context, thread *model.Thread) error {
	return r.db.WithContext(ctx).Save(thread).Error
}
