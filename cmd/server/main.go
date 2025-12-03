package main

import (
	"log"
	"os"

	"anoa.com/telkomalumiforum/internal/handler"
	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/internal/service"
	"anoa.com/telkomalumiforum/pkg/database"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	db := database.Connect()
	if err := migrate(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	if err := seedRoles(db); err != nil {
		log.Fatalf("failed to seed roles: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	imageStorage, err := storage.NewCloudinaryStorage()
	if err != nil {
		log.Fatalf("failed to initialize cloudinary storage: %v", err)
	}
	authService := service.NewAuthService(userRepo, imageStorage)
	authHandler := handler.NewAuthHandler(authService)

	router := gin.Default()

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Role{},
		&model.User{},
		&model.Profile{},
	)
}

func seedRoles(db *gorm.DB) error {
	defaultRoles := []model.Role{
		{Name: "admin", Description: "Super administrator"},
		{Name: "guru", Description: "Guru"},
		{Name: "siswa", Description: "Siswa"},
	}

	for _, role := range defaultRoles {
		var count int64
		if err := db.Model(&model.Role{}).
			Where("name = ?", role.Name).
			Count(&count).Error; err != nil {
			return err
		}

		if count == 0 {
			if err := db.Create(&role).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
