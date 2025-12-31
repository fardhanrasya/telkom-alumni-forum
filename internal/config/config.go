package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	Port           string
	AllowedOrigins string

	DatabaseURL    string
	RedisURL       string
	
	MeiliSearchHost string
	MeiliMasterKey  string

	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
	CloudinaryUploadFolder string

	RateLimitGlobal time.Duration
	RateLimitThread time.Duration
}

func Load() (*Config, error) {
	// Don't fail if .env doesn't exist (might be prod env vars)
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		Port:           getEnv("PORT", "8080"),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
		
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),

		MeiliSearchHost: getEnv("MEILISEARCH_HOST", "http://localhost:7700"),
		MeiliMasterKey:  os.Getenv("MEILI_MASTER_KEY"),

		CloudinaryCloudName:    os.Getenv("CLOUDINARY_CLOUD_NAME"),
		CloudinaryAPIKey:       os.Getenv("CLOUDINARY_API_KEY"),
		CloudinaryAPISecret:    os.Getenv("CLOUDINARY_API_SECRET"),
		CloudinaryUploadFolder: getEnv("CLOUDINARY_UPLOAD_FOLDER", "telkom_alumni_forum"),
	}

	// Parsing durations
	var err error
	cfg.RateLimitGlobal, err = parseDuration(getEnv("RATE_LIMIT_GLOBAL", "5s"))
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_GLOBAL: %w", err)
	}
	cfg.RateLimitThread, err = parseDuration(getEnv("RATE_LIMIT_THREAD", "5m"))
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_THREAD: %w", err)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
