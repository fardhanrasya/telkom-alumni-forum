package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"anoa.com/telkomalumiforum/internal/entity"
	search "anoa.com/telkomalumiforum/internal/modules/search/service"
	"anoa.com/telkomalumiforum/internal/modules/user/dto"
	"anoa.com/telkomalumiforum/internal/modules/user/repository"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

type AuthService interface {
	Login(ctx context.Context, input dto.LoginInput) (*dto.AuthResponse, error)
	GoogleLogin() string
	GoogleCallback(ctx context.Context, code string) (*dto.AuthResponse, error)
}

type authService struct {
	repo         repository.UserRepository
	imageStorage storage.ImageStorage
	secret       string
	tokenTTL     time.Duration
	defaultRole  string
	meili        search.MeiliSearchService
	googleConfig *oauth2.Config
}

func NewAuthService(repo repository.UserRepository, imageStorage storage.ImageStorage, meili search.MeiliSearchService) AuthService {
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

	googleConfig := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &authService{
		repo:         repo,
		imageStorage: imageStorage,
		secret:       secret,
		tokenTTL:     ttl,
		defaultRole:  defaultRole,
		meili:        meili,
		googleConfig: googleConfig,
	}
}

func (s *authService) Login(ctx context.Context, input dto.LoginInput) (*dto.AuthResponse, error) {
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

func (s *authService) GoogleLogin() string {
	return s.googleConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

func (s *authService) GoogleCallback(ctx context.Context, code string) (*dto.AuthResponse, error) {
	token, err := s.googleConfig.Exchange(ctx, code)
	if err != nil {
		return nil, errors.New("failed to exchange token: " + err.Error())
	}

	client := s.googleConfig.Client(ctx, token)
	userInfoResp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, errors.New("failed to get user info: " + err.Error())
	}
	defer userInfoResp.Body.Close()

	var googleUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		Picture       string `json:"picture"`
	}

	if err := json.NewDecoder(userInfoResp.Body).Decode(&googleUser); err != nil {
		return nil, errors.New("failed to decode user info: " + err.Error())
	}

	if !strings.HasSuffix(googleUser.Email, "@student.smktelkom-jkt.sch.id") {
		return nil, errors.New("email domain must be @student.smktelkom-jkt.sch.id")
	}

	user, err := s.repo.FindByEmail(ctx, googleUser.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Register new user
			randomPassword := uuid.New().String()
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)

			role, err := s.repo.FindRoleByName(ctx, s.defaultRole)
			if err != nil {
				return nil, errors.New("default role not found")
			}

			// Generate username from email (before @)
			username := strings.Split(googleUser.Email, "@")[0]
			// Replace spaces with underscores
			username = strings.ReplaceAll(username, " ", "_")

			// Check if username exists, if so append random string
			if _, err := s.repo.FindByUsername(ctx, username); err == nil {
				username = username + "_" + uuid.New().String()[:4]
			}

			newUser := &entity.User{
				Username:     username,
				Email:        googleUser.Email,
				PasswordHash: string(hashedPassword),
				RoleID:       &role.ID,
				Role:         *role,
				AvatarURL:    &googleUser.Picture,
				GoogleID:     &googleUser.ID,
			}

			newProfile := &entity.Profile{
				FullName: googleUser.Name,
				Bio:      stringPtr("Student at SMK Telkom Jakarta"),
			}

			if err := s.repo.Create(ctx, newUser, newProfile); err != nil {
				return nil, errors.New("failed to create user: " + err.Error())
			}

			user = newUser
		} else {
			return nil, err
		}
	} else {
		// User found, check/update GoogleID
		if user.GoogleID == nil || *user.GoogleID != googleUser.ID {
			user.GoogleID = &googleUser.ID
			if err := s.repo.Update(ctx, user, nil); err != nil {
				log.Printf("Failed to update GoogleID for user %s: %v", user.Email, err)
			}
		}
	}

	return s.buildAuthResponse(user)
}

func stringPtr(s string) *string {
	return &s
}

func (s *authService) buildAuthResponse(user *entity.User) (*dto.AuthResponse, error) {
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	var searchToken string
	if s.meili != nil {
		roleName := "siswa" // Default fallback
		if user.RoleID != nil {
			roleName = user.Role.Name
		}
		st, err := s.meili.GenerateSearchToken(roleName)
		if err != nil {
			log.Printf("Failed to generate search token for user %s (role %s): %v", user.Username, roleName, err)
			searchToken = ""
		} else {
			searchToken = st
		}
	}

	user.PasswordHash = ""

	return &dto.AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresAt,
		User:        user,
		Role:        &user.Role,
		Profile:     user.Profile,
		SearchToken: searchToken,
	}, nil
}

func (s *authService) generateToken(user *entity.User) (string, int64, error) {
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
