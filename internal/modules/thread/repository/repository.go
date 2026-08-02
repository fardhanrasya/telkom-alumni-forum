package thread

import (
	"context"
	"fmt"
	"strings"
	"time"

	"anoa.com/telkomalumiforum/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Create(ctx context.Context, thread *entity.Thread) error
	FindBySlug(ctx context.Context, slug string) (*entity.Thread, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Thread, error)
	FindAll(ctx context.Context, categoryID *uuid.UUID, search string, audiences []string, sortBy string, offset, limit int) ([]*entity.Thread, int64, error)
	FindFeedWithFollows(ctx context.Context, currentUserID *uuid.UUID, categoryID *uuid.UUID, search string, audiences []string, sortBy string, offset, limit int) ([]*entity.Thread, int64, map[uuid.UUID]bool, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, audiences []string, offset, limit int) ([]*entity.Thread, int64, error)
	GetTrending(ctx context.Context, limit int) ([]*entity.Thread, error)
	Update(ctx context.Context, thread *entity.Thread) error
	Delete(ctx context.Context, id uuid.UUID) error
	TrackFeedViews(ctx context.Context, userID uuid.UUID, threadIDs []uuid.UUID) error
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
		Preload("User.Equip.AvatarBorder").
		Preload("User.Equip.ThreadBg").
		Preload("User.Equip.ProfileBg").
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
		Preload("User.Equip.AvatarBorder").
		Preload("User.Equip.ThreadBg").
		Preload("User.Equip.ProfileBg").
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
		Preload("User.Equip.AvatarBorder").
		Preload("User.Equip.ThreadBg").
		Preload("User.Equip.ProfileBg").
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

func (r *repository) FindFeedWithFollows(ctx context.Context, currentUserID *uuid.UUID, categoryID *uuid.UUID, search string, audiences []string, sortBy string, offset, limit int) ([]*entity.Thread, int64, map[uuid.UUID]bool, error) {
	unseenFollowedMap := make(map[uuid.UUID]bool)

	// If guest or no current user, default to standard FindAll
	if currentUserID == nil {
		threads, total, err := r.FindAll(ctx, categoryID, search, audiences, sortBy, offset, limit)
		return threads, total, unseenFollowedMap, err
	}

	// Get IDs of users followed by current user
	var followedUserIDs []uuid.UUID
	r.db.WithContext(ctx).
		Model(&entity.UserFollow{}).
		Where("follower_id = ?", *currentUserID).
		Pluck("following_id", &followedUserIDs)

	if len(followedUserIDs) == 0 {
		threads, total, err := r.FindAll(ctx, categoryID, search, audiences, sortBy, offset, limit)
		return threads, total, unseenFollowedMap, err
	}

	// Custom feed ordering:
	// Priority 1: Threads by followed users that current user HAS NOT SEEN in feed
	// Priority 2: All other threads (including seen followed threads & general threads)
	var threads []*entity.Thread
	var total int64

	baseQuery := r.db.WithContext(ctx).
		Preload("Category").
		Preload("User").
		Preload("User.Profile").
		Preload("User.Equip.AvatarBorder").
		Preload("User.Equip.ThreadBg").
		Preload("User.Equip.ProfileBg").
		Preload("Attachments")

	if categoryID != nil {
		baseQuery = baseQuery.Where("category_id = ?", categoryID)
	}
	if search != "" {
		baseQuery = baseQuery.Where("title ILIKE ? OR content ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if len(audiences) > 0 {
		baseQuery = baseQuery.Where("audience IN ?", audiences)
	}

	countQuery := baseQuery.Session(&gorm.Session{})
	if err := countQuery.Model(&entity.Thread{}).Count(&total).Error; err != nil {
		return nil, 0, unseenFollowedMap, err
	}

	var idStrs []string
	for _, id := range followedUserIDs {
		idStrs = append(idStrs, fmt.Sprintf("'%s'", id.String()))
	}
	followedIDsSQL := strings.Join(idStrs, ",")

	orderSQL := fmt.Sprintf(
		"CASE WHEN user_id IN (%s) AND id NOT IN (SELECT thread_id FROM thread_feed_views WHERE user_id = '%s') THEN 0 ELSE 1 END, created_at DESC",
		followedIDsSQL,
		currentUserID.String(),
	)

	if sortBy == "popular" {
		orderSQL = fmt.Sprintf(
			"CASE WHEN user_id IN (%s) AND id NOT IN (SELECT thread_id FROM thread_feed_views WHERE user_id = '%s') THEN 0 ELSE 1 END, views DESC, created_at DESC",
			followedIDsSQL,
			currentUserID.String(),
		)
	}

	if err := baseQuery.Order(orderSQL).Offset(offset).Limit(limit).Find(&threads).Error; err != nil {
		return nil, 0, unseenFollowedMap, err
	}

	// Identify which returned threads are unseen followed threads
	var unseenIDs []uuid.UUID
	r.db.WithContext(ctx).
		Model(&entity.Thread{}).
		Where("user_id IN ? AND id NOT IN (SELECT thread_id FROM thread_feed_views WHERE user_id = ?)", followedUserIDs, *currentUserID).
		Pluck("id", &unseenIDs)

	for _, id := range unseenIDs {
		unseenFollowedMap[id] = true
	}

	return threads, total, unseenFollowedMap, nil
}

func (r *repository) FindByUserID(ctx context.Context, userID uuid.UUID, audiences []string, offset, limit int) ([]*entity.Thread, int64, error) {
	var threads []*entity.Thread
	var total int64

	query := r.db.WithContext(ctx).
		Preload("Category").
		Preload("User").
		Preload("User.Profile").
		Preload("User.Equip.AvatarBorder").
		Preload("User.Equip.ThreadBg").
		Preload("User.Equip.ProfileBg").
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

func (r *repository) TrackFeedViews(ctx context.Context, userID uuid.UUID, threadIDs []uuid.UUID) error {
	if len(threadIDs) == 0 {
		return nil
	}

	now := time.Now()
	var feedViews []entity.ThreadFeedView
	for _, tid := range threadIDs {
		feedViews = append(feedViews, entity.ThreadFeedView{
			UserID:   userID,
			ThreadID: tid,
			ViewedAt: now,
		})
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&feedViews).Error
}
