package database

import (
	"fmt"
	"log"
	"os"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB   *gorm.DB
	once sync.Once
)

func Connect() *gorm.DB {
	once.Do(func() {
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			valueOrDefault("DB_HOST", "localhost"),
			valueOrDefault("DB_USER", "postgres"),
			os.Getenv("DB_PASS"),
			valueOrDefault("DB_NAME", "telkom_forum"),
			valueOrDefault("DB_PORT", "5432"),
		)

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect database: %v", err)
		}

		DB = db
	})

	return DB
}

func GetDB() *gorm.DB {
	if DB == nil {
		return Connect()
	}
	return DB
}

func valueOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return fallback
}
