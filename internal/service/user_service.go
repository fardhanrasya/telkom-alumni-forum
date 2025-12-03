package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterInput struct {
	Username       string  `json:"username" form:"username" binding:"required,min=3,max=50"`
	Email          string  `json:"email" form:"email" binding:"required,email"`
	Password       string  `json:"password" form:"password" binding:"required,min=8"`
	Role           *string `json:"role" form:"role"`
	FullName       string  `json:"full_name" form:"full_name" binding:"required"`
	IdentityNumber *string `json:"identity_number" form:"identity_number"`
	ClassGrade     *string `json:"class_grade" form:"class_grade"`
	Bio            *string `json:"bio" form:"bio"`
}

// AvatarFile merepresentasikan file avatar yang diupload user.
type AvatarFile struct {
	Reader   io.Reader
	FileName string
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	AccessToken string         `json:"access_token"`
	TokenType   string         `json:"token_type"`
	ExpiresIn   int64          `json:"expires_in"`
	User        *model.User    `json:"user"`
	Role        *model.Role    `json:"role"`
	Profile     *model.Profile `json:"profile"`
}

type AuthService interface {
	Register(ctx context.Context, input RegisterInput, avatar *AvatarFile) (*AuthResponse, error)
	Login(ctx context.Context, input LoginInput) (*AuthResponse, error)
}

type authService struct {
	repo         repository.UserRepository
	imageStorage storage.ImageStorage
	secret       string
	tokenTTL     time.Duration
	defaultRole  string
}

func NewAuthService(repo repository.UserRepository, imageStorage storage.ImageStorage) AuthService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "change-me"
	}

	ttl := time.Hour
	if ttlStr := os.Getenv("JWT_TTL_MINUTES"); ttlStr != "" {
		if minutes, err := strconv.Atoi(ttlStr); err == nil {
			ttl = time.Duration(minutes) * time.Minute
		}
	}

	defaultRole := os.Getenv("DEFAULT_ROLE")
	if defaultRole == "" {
		defaultRole = "siswa"
	}

	return &authService{
		repo:         repo,
		imageStorage: imageStorage,
		secret:       secret,
		tokenTTL:     ttl,
		defaultRole:  defaultRole,
	}
}

func (s *authService) Register(ctx context.Context, input RegisterInput, avatar *AvatarFile) (*AuthResponse, error) {
	if err := s.ensureUserUnique(ctx, input.Email, input.Username); err != nil {
		return nil, err
	}

	roleName := s.defaultRole
	if input.Role != nil && *input.Role != "" {
		roleName = *input.Role
	}

	role, err := s.repo.FindRoleByName(ctx, roleName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("role %s not found", roleName)
		}
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Upload avatar (jika ada) setelah semua validasi bisnis (role, user unik) lolos.
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

	return s.buildAuthResponse(createdUser)
}

func (s *authService) Login(ctx context.Context, input LoginInput) (*AuthResponse, error) {
	user, err := s.repo.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return s.buildAuthResponse(user)
}

func (s *authService) buildAuthResponse(user *model.User) (*AuthResponse, error) {
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = ""

	return &AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresAt,
		User:        user,
		Role:        &user.Role,
		Profile:     user.Profile,
	}, nil
}

func (s *authService) generateToken(user *model.User) (string, int64, error) {
	expiresAt := time.Now().Add(s.tokenTTL)

	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", user.ID),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return "", 0, err
	}

	return signed, expiresAt.Unix(), nil
}

func (s *authService) ensureUserUnique(ctx context.Context, email, username string) error {
	if _, err := s.repo.FindByEmail(ctx, email); err == nil {
		return errors.New("email already registered")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if _, err := s.repo.FindByUsername(ctx, username); err == nil {
		return errors.New("username already taken")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return nil
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
