package repository

import (
	"context"

	"anoa.com/telkomalumiforum/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostRepository interface {
	Create(ctx context.Context, post *model.Post) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Post, error)
	FindByThreadID(ctx context.Context, threadID uuid.UUID, offset, limit int) ([]*model.Post, int64, error)
	FindAllByThreadID(ctx context.Context, threadID uuid.UUID) ([]*model.Post, error)
	Update(ctx context.Context, post *model.Post) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(ctx context.Context, post *model.Post) error {
	return r.db.WithContext(ctx).Create(post).Error
}

func (r *postRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Post, error) {
	var post model.Post
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("User.Profile").
		Preload("Attachments").
		Preload("Parent"). 
		Where("id = ?", id).
		First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *postRepository) FindByThreadID(ctx context.Context, threadID uuid.UUID, offset, limit int) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var total int64
	
	query := r.db.WithContext(ctx).
		Preload("User").
		Preload("User.Profile").
		Preload("Attachments").
		Where("thread_id = ?", threadID)

	if err := query.Model(&model.Post{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at ASC").Offset(offset).Limit(limit).Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

func (r *postRepository) FindAllByThreadID(ctx context.Context, threadID uuid.UUID) ([]*model.Post, error) {
	var posts []*model.Post
	
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("User.Profile").
		Preload("Attachments").
		Where("thread_id = ?", threadID).
		Order("created_at ASC").
		Find(&posts).Error
		
	return posts, err
}

func (r *postRepository) Update(ctx context.Context, post *model.Post) error {
	return r.db.WithContext(ctx).Save(post).Error
}

func (r *postRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Post{}, id).Error
}
