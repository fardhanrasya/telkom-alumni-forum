package service

import (
	"context"
	"errors"
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
		Subject:   user.ID.String(),
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
