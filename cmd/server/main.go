package main

import (
	"log"
	"os"

	"anoa.com/telkomalumiforum/internal/handler"
	"anoa.com/telkomalumiforum/internal/middleware"
	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/internal/service"
	"anoa.com/telkomalumiforum/pkg/database"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
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

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "development" {
		if err := seedAdminUser(db); err != nil {
			log.Fatalf("failed to seed admin user: %v", err)
		}
	}

	userRepo := repository.NewUserRepository(db)
	imageStorage, err := storage.NewCloudinaryStorage()
	if err != nil {
		log.Fatalf("failed to initialize cloudinary storage: %v", err)
	}

	authService := service.NewAuthService(userRepo, imageStorage)
	authHandler := handler.NewAuthHandler(authService)

	adminService := service.NewAdminService(userRepo, imageStorage)
	adminHandler := handler.NewAdminHandler(adminService)

	profileService := service.NewProfileService(userRepo, imageStorage)
	profileHandler := handler.NewProfileHandler(profileService)

	router := gin.Default()

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		auth.POST("/login", authHandler.Login)
	}

	authMiddleware := middleware.NewAuthMiddleware(userRepo)

	api.Use(authMiddleware.RequireAuth())
	{
		admin := api.Group("/admin")
		admin.Use(authMiddleware.RequireAdmin())
		{
			admin.POST("/users", adminHandler.CreateUser)
		}

		profile := api.Group("/profile")
		{
			profile.PUT("", profileHandler.UpdateProfile)
		}
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

func seedAdminUser(db *gorm.DB) error {
	var adminRole model.Role
	if err := db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}

	var count int64
	if err := db.Model(&model.User{}).
		Where("email = ?", "admin@telkom.com").
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Println("Admin user already exists, skipping seed")
		return nil
	}

	password := "admin123"
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	adminUser := model.User{
		Username:     "admin",
		Email:        "admin@telkom.com",
		PasswordHash: string(hashedPasswordBytes),
		RoleID:       &adminRole.ID,
	}

	if err := db.Create(&adminUser).Error; err != nil {
		return err
	}

	adminProfile := model.Profile{
		UserID:   adminUser.ID,
		FullName: "Administrator",
		Bio:      stringPtr("System Administrator"),
	}

	if err := db.Create(&adminProfile).Error; err != nil {
		return err
	}

	log.Println("✅ Admin user seeded successfully")
	log.Println("   Email: admin@telkom.com")
	log.Println("   Password: admin123")

	return nil
}

func stringPtr(s string) *string {
	return &s
}
