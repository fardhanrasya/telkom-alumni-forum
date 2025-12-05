package service

import (
	"context"
	"errors"
	"fmt"

	"anoa.com/telkomalumiforum/internal/dto"
	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/pkg/storage"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminService interface {
	CreateUser(ctx context.Context, input dto.CreateUserInput, avatar *dto.AvatarFile) (*dto.CreateUserResponse, error)
}

type adminService struct {
	repo         repository.UserRepository
	imageStorage storage.ImageStorage
}

func NewAdminService(repo repository.UserRepository, imageStorage storage.ImageStorage) AdminService {
	return &adminService{
		repo:         repo,
		imageStorage: imageStorage,
	}
}

func (s *adminService) CreateUser(ctx context.Context, input dto.CreateUserInput, avatar *dto.AvatarFile) (*dto.CreateUserResponse, error) {
	if _, err := s.repo.FindByEmail(ctx, input.Email); err == nil {
		return nil, errors.New("email already registered")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if _, err := s.repo.FindByUsername(ctx, input.Username); err == nil {
		return nil, errors.New("username already taken")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	role, err := s.repo.FindRoleByName(ctx, input.Role)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("role %s not found", input.Role)
		}
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	var avatarURL *string
	if avatar != nil && avatar.Reader != nil && s.imageStorage != nil {
		url, err := s.imageStorage.UploadImage(ctx, avatar.Reader, "avatars", avatar.FileName)
		if err != nil {
			return nil, err
		}
		avatarURL = &url
	}

	roleID := role.ID
	user := &model.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		RoleID:       &roleID,
		AvatarURL:    avatarURL,
	}

	profile := &model.Profile{
		FullName:       input.FullName,
		IdentityNumber: normalizeOptional(input.IdentityNumber),
		ClassGrade:     normalizeOptional(input.ClassGrade),
		Bio:            normalizeOptional(input.Bio),
	}

	if err := s.repo.Create(ctx, user, profile); err != nil {
		return nil, err
	}

	createdUser, err := s.repo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}

	createdUser.PasswordHash = ""

	return &dto.CreateUserResponse{
		User:    createdUser,
		Role:    &createdUser.Role,
		Profile: createdUser.Profile,
	}, nil
}
