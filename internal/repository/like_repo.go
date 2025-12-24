package repository

import (
	"context"

	"anoa.com/telkomalumiforum/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LikeRepository interface {
	LikeThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error
	UnlikeThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error
	LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
	UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
	IsThreadLiked(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) (bool, error)
	IsPostLiked(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error)
}

type likeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) LikeRepository {
	return &likeRepository{db: db}
}

func (r *likeRepository) LikeThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error {
	like := model.ThreadLike{
		UserID:   userID,
		ThreadID: threadID,
	}
	// Use Clause to ignore duplicate key error just in case
	return r.db.WithContext(ctx).Clauses().Create(&like).Error
	// Or standard Create, if duplicate it returns error, which is fine
	// return r.db.WithContext(ctx).Create(&like).Error
}

func (r *likeRepository) UnlikeThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		Delete(&model.ThreadLike{}).Error
}

func (r *likeRepository) LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	like := model.PostLike{
		UserID: userID,
		PostID: postID,
	}
	return r.db.WithContext(ctx).Create(&like).Error
}

func (r *likeRepository) UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(&model.PostLike{}).Error
}

func (r *likeRepository) IsThreadLiked(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ThreadLike{}).
		Where("user_id = ? AND thread_id = ?", userID, threadID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *likeRepository) IsPostLiked(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.PostLike{}).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
