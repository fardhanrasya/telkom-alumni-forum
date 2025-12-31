package service

import (
	"context"
	"encoding/json"
	"fmt"

	"anoa.com/telkomalumiforum/internal/entity"
	notifRepo "anoa.com/telkomalumiforum/internal/modules/notification/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type NotificationService interface {
	CreateNotification(ctx context.Context, notification *entity.Notification) error
	GetNotifications(userID uuid.UUID, limit, offset int) ([]entity.Notification, error)
	MarkAsRead(id uuid.UUID) error
	MarkAllAsRead(userID uuid.UUID) error
	UnreadCount(userID uuid.UUID) (int64, error)
}

type notificationService struct {
	repo        notifRepo.NotificationRepository
	redisClient *redis.Client
}

func NewNotificationService(repo notifRepo.NotificationRepository, redisClient *redis.Client) NotificationService {
	return &notificationService{
		repo:        repo,
		redisClient: redisClient,
	}
}

func (s *notificationService) CreateNotification(ctx context.Context, notification *entity.Notification) error {
	// 1. Save to DB
	if err := s.repo.Create(notification); err != nil {
		return err
	}

	// 2. Publish to Redis if Redis is available
	if s.redisClient != nil {
		channel := fmt.Sprintf("user_notifications:%s", notification.UserID.String())

		payload, err := json.Marshal(notification)
		if err == nil {
			s.redisClient.Publish(ctx, channel, payload)
		}
	}

	return nil
}

func (s *notificationService) GetNotifications(userID uuid.UUID, limit, offset int) ([]entity.Notification, error) {
	return s.repo.GetByUserID(userID, limit, offset)
}

func (s *notificationService) MarkAsRead(id uuid.UUID) error {
	return s.repo.MarkAsRead(id)
}

func (s *notificationService) MarkAllAsRead(userID uuid.UUID) error {
	return s.repo.MarkAllAsRead(userID)
}

func (s *notificationService) UnreadCount(userID uuid.UUID) (int64, error) {
	return s.repo.CountUnread(userID)
}
