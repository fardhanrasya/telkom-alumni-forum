package repository

import (
	"anoa.com/telkomalumiforum/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepository interface {
	Create(notification *model.Notification) error
	GetByUserID(userID uuid.UUID, limit, offset int) ([]model.Notification, error)
	MarkAsRead(id uuid.UUID) error
	MarkAllAsRead(userID uuid.UUID) error
	CountUnread(userID uuid.UUID) (int64, error)
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(notification *model.Notification) error {
	return r.db.Create(notification).Error
}

func (r *notificationRepository) GetByUserID(userID uuid.UUID, limit, offset int) ([]model.Notification, error) {
	var notifications []model.Notification
	err := r.db.Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Preload("Actor", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "username", "avatar_url")
		}).
		Find(&notifications).Error
	return notifications, err
}

func (r *notificationRepository) MarkAsRead(id uuid.UUID) error {
	return r.db.Model(&model.Notification{}).Where("id = ?", id).Update("is_read", true).Error
}

func (r *notificationRepository) MarkAllAsRead(userID uuid.UUID) error {
	return r.db.Model(&model.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Update("is_read", true).Error
}

func (r *notificationRepository) CountUnread(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&model.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count).Error
	return count, err
}
