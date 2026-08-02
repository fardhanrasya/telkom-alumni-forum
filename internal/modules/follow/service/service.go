package service

import (
	"context"
	"errors"
	"fmt"

	"anoa.com/telkomalumiforum/internal/entity"
	followRepo "anoa.com/telkomalumiforum/internal/modules/follow/repository"
	missionService "anoa.com/telkomalumiforum/internal/modules/mission/service"
	notifService "anoa.com/telkomalumiforum/internal/modules/notification/service"
	userRepo "anoa.com/telkomalumiforum/internal/modules/user/repository"
	"github.com/google/uuid"
)

type FollowStatus struct {
	IsFollowing    bool  `json:"is_following"`
	FollowersCount int64 `json:"followers_count"`
	FollowingCount int64 `json:"following_count"`
}

type FollowService interface {
	ToggleFollow(ctx context.Context, followerIDStr string, targetUsername string) (bool, error)
	GetFollowStatus(ctx context.Context, currentUserIDStr string, targetUsername string) (*FollowStatus, error)
}

type followService struct {
	followRepo     followRepo.FollowRepository
	userRepo       userRepo.UserRepository
	notifSvc       notifService.NotificationService
	missionService missionService.MissionService
}

func NewFollowService(
	followRepo followRepo.FollowRepository,
	userRepo userRepo.UserRepository,
	notifSvc notifService.NotificationService,
	missionSvc missionService.MissionService,
) FollowService {
	return &followService{
		followRepo:     followRepo,
		userRepo:       userRepo,
		notifSvc:       notifSvc,
		missionService: missionSvc,
	}
}

func (s *followService) ToggleFollow(ctx context.Context, followerIDStr string, targetUsername string) (bool, error) {
	followerID, err := uuid.Parse(followerIDStr)
	if err != nil {
		return false, errors.New("invalid follower user ID")
	}

	targetUser, err := s.userRepo.FindByUsername(ctx, targetUsername)
	if err != nil {
		return false, errors.New("target user not found")
	}

	if followerID == targetUser.ID {
		return false, errors.New("you cannot follow yourself")
	}

	isFollowing, err := s.followRepo.IsFollowing(ctx, followerID, targetUser.ID)
	if err != nil {
		return false, err
	}

	if isFollowing {
		if err := s.followRepo.Unfollow(ctx, followerID, targetUser.ID); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := s.followRepo.Follow(ctx, followerID, targetUser.ID); err != nil {
		return false, err
	}

	if s.missionService != nil {
		s.missionService.RecordProgressAsync(followerID, "follow", 1)
	}

	// Trigger Notification for new follower
	if s.notifSvc != nil {
		followerUser, err := s.userRepo.FindByID(ctx, followerIDStr)
		if err == nil && followerUser != nil {
			notif := &entity.Notification{
				UserID:     targetUser.ID,
				ActorID:    followerID,
				EntityID:   targetUser.ID,
				EntityType: "user",
				Type:       "follow",
				Message:    fmt.Sprintf("%s mulai mengikuti Anda", followerUser.Username),
				IsRead:     false,
			}
			_ = s.notifSvc.CreateNotification(ctx, notif)
		}
	}

	return true, nil
}

func (s *followService) GetFollowStatus(ctx context.Context, currentUserIDStr string, targetUsername string) (*FollowStatus, error) {
	targetUser, err := s.userRepo.FindByUsername(ctx, targetUsername)
	if err != nil {
		return nil, errors.New("target user not found")
	}

	followersCount, err := s.followRepo.GetFollowersCount(ctx, targetUser.ID)
	if err != nil {
		return nil, err
	}

	followingCount, err := s.followRepo.GetFollowingCount(ctx, targetUser.ID)
	if err != nil {
		return nil, err
	}

	isFollowing := false
	if currentUserIDStr != "" {
		if currentID, err := uuid.Parse(currentUserIDStr); err == nil {
			isFollowing, _ = s.followRepo.IsFollowing(ctx, currentID, targetUser.ID)
		}
	}

	return &FollowStatus{
		IsFollowing:    isFollowing,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
	}, nil
}
