package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"anoa.com/telkomalumiforum/internal/dto"
	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/pkg/storage"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ProfileService interface {
	UpdateProfile(ctx context.Context, userID string, input dto.UpdateProfileInput, avatar *dto.AvatarFile) (*dto.UpdateProfileResponse, error)
	GetProfileByUsername(ctx context.Context, username string) (*dto.PublicProfileResponse, error)
	GetCurrentProfile(ctx context.Context, userID string) (*dto.UpdateProfileResponse, error)
}

type profileService struct {
	repo            repository.UserRepository
	imageStorage    storage.ImageStorage
	leaderboardRepo repository.LeaderboardRepository
}

func NewProfileService(repo repository.UserRepository, imageStorage storage.ImageStorage, leaderboardRepo repository.LeaderboardRepository) ProfileService {
	return &profileService{
		repo:            repo,
		imageStorage:    imageStorage,
		leaderboardRepo: leaderboardRepo,
	}
}

func (s *profileService) UpdateProfile(ctx context.Context, userID string, input dto.UpdateProfileInput, avatar *dto.AvatarFile) (*dto.UpdateProfileResponse, error) {
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

	var profile *model.Profile
	if user.Profile != nil {
		profile = user.Profile
		if input.Bio != nil {
			profile.Bio = normalizeOptional(input.Bio)
		}
	}

	if err := s.repo.Update(ctx, user, profile); err != nil {
		return nil, err
	}

	updatedUser, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	updatedUser.PasswordHash = ""

	// Get gamification stats
	var allTimeScore, weeklyScore int
	if s.leaderboardRepo != nil {
		stats, err := s.leaderboardRepo.GetUserStatsByUserID(updatedUser.ID)
		if err == nil && stats != nil {
			allTimeScore = stats.TotalScoreAllTime
			weeklyScore = stats.TotalScoreWeekly
		}
	}
	gamificationStatus := GetGamificationStatusWithWeekly(allTimeScore, weeklyScore)

	return &dto.UpdateProfileResponse{
		User:    updatedUser,
		Profile: updatedUser.Profile,
		GamificationStatus: dto.GamificationStatus{
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

func (s *profileService) GetProfileByUsername(ctx context.Context, username string) (*dto.PublicProfileResponse, error) {
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

	// Calculate gamification status
	gamificationStatus := GetGamificationStatusWithWeekly(allTimeScore, weeklyScore)

	response := &dto.PublicProfileResponse{
		Username:  user.Username,
		Role:      user.Role.Name,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt,
		GamificationStatus: dto.GamificationStatus{
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
		response.ClassGrade = user.Profile.ClassGrade
		response.Bio = user.Profile.Bio
	}

	return response, nil
}

func (s *profileService) GetCurrentProfile(ctx context.Context, userID string) (*dto.UpdateProfileResponse, error) {
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
	gamificationStatus := GetGamificationStatusWithWeekly(allTimeScore, weeklyScore)

	return &dto.UpdateProfileResponse{
		User:    user,
		Profile: user.Profile,
		GamificationStatus: dto.GamificationStatus{
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
