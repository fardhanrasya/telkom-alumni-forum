package repository

import (
	"context"

	"anoa.com/telkomalumiforum/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FollowRepository interface {
	Follow(ctx context.Context, followerID, followingID uuid.UUID) error
	Unfollow(ctx context.Context, followerID, followingID uuid.UUID) error
	IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error)
	GetFollowersCount(ctx context.Context, userID uuid.UUID) (int64, error)
	GetFollowingCount(ctx context.Context, userID uuid.UUID) (int64, error)
	GetFollowingIDs(ctx context.Context, followerID uuid.UUID) ([]uuid.UUID, error)
}

type followRepository struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &followRepository{db: db}
}

func (r *followRepository) Follow(ctx context.Context, followerID, followingID uuid.UUID) error {
	follow := &entity.UserFollow{
		FollowerID:  followerID,
		FollowingID: followingID,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(follow).Error
}

func (r *followRepository) Unfollow(ctx context.Context, followerID, followingID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Delete(&entity.UserFollow{}).Error
}

func (r *followRepository) IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.UserFollow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&count).Error
	return count > 0, err
}

func (r *followRepository) GetFollowersCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.UserFollow{}).
		Where("following_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *followRepository) GetFollowingCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.UserFollow{}).
		Where("follower_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *followRepository) GetFollowingIDs(ctx context.Context, followerID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).
		Model(&entity.UserFollow{}).
		Where("follower_id = ?", followerID).
		Pluck("following_id", &ids).Error
	return ids, err
}
