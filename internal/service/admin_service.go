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
	GetAllUsers(ctx context.Context) ([]*dto.AdminUserResponse, error)
	DeleteUser(ctx context.Context, id string) error
	UpdateUser(ctx context.Context, id string, input dto.UpdateAdminUserInput, avatar *dto.AvatarFile) (*dto.AdminUserResponse, error)
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
		Angkatan:       normalizeOptional(input.Angkatan),
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

func (s *adminService) GetAllUsers(ctx context.Context) ([]*dto.AdminUserResponse, error) {
	users, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	var response []*dto.AdminUserResponse
	for _, u := range users {
		u.PasswordHash = ""
		response = append(response, &dto.AdminUserResponse{
			User:    u,
			Role:    &u.Role,
			Profile: u.Profile,
		})
	}

	return response, nil
}

func (s *adminService) DeleteUser(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *adminService) UpdateUser(ctx context.Context, id string, input dto.UpdateAdminUserInput, avatar *dto.AvatarFile) (*dto.AdminUserResponse, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update User fields
	if input.Username != "" && input.Username != user.Username {
		if _, err := s.repo.FindByUsername(ctx, input.Username); err == nil {
			return nil, errors.New("username already taken")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		user.Username = input.Username
	}

	if input.Email != "" && input.Email != user.Email {
		if _, err := s.repo.FindByEmail(ctx, input.Email); err == nil {
			return nil, errors.New("email already registered")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		user.Email = input.Email
	}

	if input.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = string(hashedPassword)
	}

	// Update Role
	if input.Role != "" {
		if user.Role.Name != input.Role {
			role, err := s.repo.FindRoleByName(ctx, input.Role)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, fmt.Errorf("role %s not found", input.Role)
				}
				return nil, err
			}
			user.RoleID = &role.ID
			user.Role = *role
		}
	}

	// Upload new Avatar
	if avatar != nil && avatar.Reader != nil && s.imageStorage != nil {
		url, err := s.imageStorage.UploadImage(ctx, avatar.Reader, "avatars", avatar.FileName)
		if err != nil {
			return nil, err
		}
		user.AvatarURL = &url
	}

	// Update Profile
	if user.Profile == nil {
		user.Profile = &model.Profile{UserID: user.ID}
	}

	if input.FullName != "" {
		user.Profile.FullName = input.FullName
	}

	// Optional fields logic: if not nil, update
	if input.IdentityNumber != nil {
		user.Profile.IdentityNumber = normalizeOptional(input.IdentityNumber)
	}
	if input.Angkatan != nil {
		user.Profile.Angkatan = normalizeOptional(input.Angkatan)
	}
	if input.Bio != nil {
		user.Profile.Bio = normalizeOptional(input.Bio)
	}

	if err := s.repo.Update(ctx, user, user.Profile); err != nil {
		return nil, err
	}

	// Refresh user data (or just use what we have, but cleaner to return what's in DB mainly for timestamps or if triggers affected it, but here we can just return what we have)
	// To be safe and because FindByID preloads everything nicely:
	updatedUser, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	updatedUser.PasswordHash = ""

	return &dto.AdminUserResponse{
		User:    updatedUser,
		Role:    &updatedUser.Role,
		Profile: updatedUser.Profile,
	}, nil
}
