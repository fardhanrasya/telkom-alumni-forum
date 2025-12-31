package thread

import (
	"context"

	"anoa.com/telkomalumiforum/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, thread *entity.Thread) error
	FindBySlug(ctx context.Context, slug string) (*entity.Thread, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Thread, error)
	FindAll(ctx context.Context, categoryID *uuid.UUID, search string, audiences []string, sortBy string, offset, limit int) ([]*entity.Thread, int64, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, audiences []string, offset, limit int) ([]*entity.Thread, int64, error)
	GetTrending(ctx context.Context, limit int) ([]*entity.Thread, error)
	Update(ctx context.Context, thread *entity.Thread) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, thread *entity.Thread) error {
	return r.db.WithContext(ctx).Create(thread).Error
}

func (r *repository) FindBySlug(ctx context.Context, slug string) (*entity.Thread, error) {
	var thread entity.Thread
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

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Thread, error) {
	var thread entity.Thread
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

func (r *repository) FindAll(ctx context.Context, categoryID *uuid.UUID, search string, audiences []string, sortBy string, offset, limit int) ([]*entity.Thread, int64, error) {
	var threads []*entity.Thread
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

	if err := query.Model(&entity.Thread{}).Count(&total).Error; err != nil {
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

func (r *repository) FindByUserID(ctx context.Context, userID uuid.UUID, audiences []string, offset, limit int) ([]*entity.Thread, int64, error) {
	var threads []*entity.Thread
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

	if err := query.Model(&entity.Thread{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&threads).Error; err != nil {
		return nil, 0, err
	}

	return threads, total, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.Thread{}, id).Error
}

func (r *repository) Update(ctx context.Context, thread *entity.Thread) error {
	return r.db.WithContext(ctx).Save(thread).Error
}
