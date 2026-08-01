package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"anoa.com/telkomalumiforum/internal/entity"
	followRepo "anoa.com/telkomalumiforum/internal/modules/follow/repository"
	leaderboardRepo "anoa.com/telkomalumiforum/internal/modules/leaderboard/repository"
	leaderboard "anoa.com/telkomalumiforum/internal/modules/leaderboard/service"
	profileDto "anoa.com/telkomalumiforum/internal/modules/profile/dto"
	userRepo "anoa.com/telkomalumiforum/internal/modules/user/repository"
	commonDto "anoa.com/telkomalumiforum/pkg/dto"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ProfileService interface {
	UpdateProfile(ctx context.Context, userID string, input profileDto.UpdateProfileInput, avatar *commonDto.AvatarFile) (*profileDto.UpdateProfileResponse, error)
	GetProfileByUsername(ctx context.Context, currentUserIDStr string, username string) (*profileDto.PublicProfileResponse, error)
	GetCurrentProfile(ctx context.Context, userID string) (*profileDto.UpdateProfileResponse, error)
}

type profileService struct {
	repo            userRepo.UserRepository
	imageStorage    storage.ImageStorage
	leaderboardRepo leaderboardRepo.LeaderboardRepository
	followRepo      followRepo.FollowRepository
}

func NewProfileService(
	repo userRepo.UserRepository,
	imageStorage storage.ImageStorage,
	leaderboardRepo leaderboardRepo.LeaderboardRepository,
	followRepo followRepo.FollowRepository,
) ProfileService {
	return &profileService{
		repo:            repo,
		imageStorage:    imageStorage,
		leaderboardRepo: leaderboardRepo,
		followRepo:      followRepo,
	}
}

func (s *profileService) UpdateProfile(ctx context.Context, userID string, input profileDto.UpdateProfileInput, avatar *commonDto.AvatarFile) (*profileDto.UpdateProfileResponse, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if input.Username != nil && *input.Username != "" && *input.Username != user.Username {
		sanitizedUsername := strings.ReplaceAll(*input.Username, " ", "_")
		if len(sanitizedUsername) < 3 {
			return nil, errors.New("username minimal 3 karakter")
		}
		if len(sanitizedUsername) > 50 {
			return nil, errors.New("username maksimal 50 karakter")
		}
		if _, err := s.repo.FindByUsername(ctx, sanitizedUsername); err == nil {
			return nil, errors.New("username already taken")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		user.Username = sanitizedUsername
	}

	if input.Password != nil && *input.Password != "" {
		if len(*input.Password) < 8 {
			return nil, errors.New("password minimal 8 karakter")
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = string(hashedPassword)
	}

	if avatar != nil && avatar.Reader != nil && s.imageStorage != nil {
		url, err := s.imageStorage.UploadImage(ctx, avatar.Reader, "avatars", avatar.FileName)
		if err != nil {
			return nil, err
		}
		user.AvatarURL = &url
	}

	var profile *entity.Profile
	if user.Profile != nil {
		profile = user.Profile
		if input.Bio != nil {
			profile.Bio = normalizeOptional(input.Bio)
		}
	}

	if err := s.repo.Update(ctx, user, profile); err != nil {
		return nil, err
	}

	return s.GetCurrentProfile(ctx, userID)
}

func (s *profileService) GetProfileByUsername(ctx context.Context, currentUserIDStr string, username string) (*profileDto.PublicProfileResponse, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Get gamification stats
	var allTimeScore, weeklyScore int
	if s.leaderboardRepo != nil {
		stats, err := s.leaderboardRepo.GetUserStatsByUserID(user.ID)
		if err == nil && stats != nil {
			allTimeScore = stats.TotalScoreAllTime
			weeklyScore = stats.TotalScoreWeekly
		}
	}

	gamificationStatus := leaderboard.GetGamificationStatusWithWeekly(allTimeScore, weeklyScore)

	var followersCount, followingCount int64
	isFollowing := false
	if s.followRepo != nil {
		followersCount, _ = s.followRepo.GetFollowersCount(ctx, user.ID)
		followingCount, _ = s.followRepo.GetFollowingCount(ctx, user.ID)

		if currentUserIDStr != "" {
			if currentID, err := uuid.Parse(currentUserIDStr); err == nil {
				isFollowing, _ = s.followRepo.IsFollowing(ctx, currentID, user.ID)
			}
		}
	}

	response := &profileDto.PublicProfileResponse{
		Username:       user.Username,
		Role:           user.Role.Name,
		AvatarURL:      user.AvatarURL,
		CreatedAt:      user.CreatedAt,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		IsFollowing:    isFollowing,
		GamificationStatus: commonDto.GamificationStatus{
			RankName:      gamificationStatus.RankName,
			NextRank:      gamificationStatus.NextRank,
			CurrentPoints: gamificationStatus.CurrentPoints,
			TargetPoints:  gamificationStatus.TargetPoints,
			Progress:      gamificationStatus.Progress,
			WeeklyPoints:  gamificationStatus.WeeklyPoints,
			WeeklyLabel:   gamificationStatus.WeeklyLabel,
		},
	}

	if user.Profile != nil {
		response.Angkatan = user.Profile.Angkatan
		response.Bio = user.Profile.Bio
	}

	return response, nil
}

func (s *profileService) GetCurrentProfile(ctx context.Context, userID string) (*profileDto.UpdateProfileResponse, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	user.PasswordHash = ""

	// Get gamification stats
	var allTimeScore, weeklyScore int
	if s.leaderboardRepo != nil {
		stats, err := s.leaderboardRepo.GetUserStatsByUserID(user.ID)
		if err == nil && stats != nil {
			allTimeScore = stats.TotalScoreAllTime
			weeklyScore = stats.TotalScoreWeekly
		}
	}
	gamificationStatus := leaderboard.GetGamificationStatusWithWeekly(allTimeScore, weeklyScore)

	var followersCount, followingCount int64
	if s.followRepo != nil {
		followersCount, _ = s.followRepo.GetFollowersCount(ctx, user.ID)
		followingCount, _ = s.followRepo.GetFollowingCount(ctx, user.ID)
	}

	return &profileDto.UpdateProfileResponse{
		User:           user,
		Profile:        user.Profile,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		GamificationStatus: commonDto.GamificationStatus{
			RankName:      gamificationStatus.RankName,
			NextRank:      gamificationStatus.NextRank,
			CurrentPoints: gamificationStatus.CurrentPoints,
			TargetPoints:  gamificationStatus.TargetPoints,
			Progress:      gamificationStatus.Progress,
			WeeklyPoints:  gamificationStatus.WeeklyPoints,
			WeeklyLabel:   gamificationStatus.WeeklyLabel,
		},
	}, nil
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	result := trimmed
	return &result
}
