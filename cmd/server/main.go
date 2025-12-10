package main

import (
	"context"
	"log"
	"os"
	"time"

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

	categoryRepo := repository.NewCategoryRepository(db)
	categoryService := service.NewCategoryService(categoryRepo)
	categoryHandler := handler.NewCategoryHandler(categoryService)

	attachmentRepo := repository.NewAttachmentRepository(db)
	attachmentService := service.NewAttachmentService(attachmentRepo, imageStorage)
	attachmentHandler := handler.NewAttachmentHandler(attachmentService)

	threadRepo := repository.NewThreadRepository(db)
	threadService := service.NewThreadService(threadRepo, categoryRepo, userRepo, attachmentRepo, imageStorage)
	threadHandler := handler.NewThreadHandler(threadService)

	postRepo := repository.NewPostRepository(db)
	postService := service.NewPostService(postRepo, threadRepo, userRepo, attachmentRepo, imageStorage)
	postHandler := handler.NewPostHandler(postService)

	router := gin.Default()

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		auth.POST("/login", authHandler.Login)
	}

	authMiddleware := middleware.NewAuthMiddleware(userRepo)

	// Public routes (tidak perlu auth)
	

	// Protected routes (perlu auth)
	api.Use(authMiddleware.RequireAuth())
	{
		admin := api.Group("/admin")
		admin.Use(authMiddleware.RequireAdmin())
		{
			admin.POST("/users", adminHandler.CreateUser)
			admin.GET("/users", adminHandler.GetAllUsers)
			admin.PUT("/users/:id", adminHandler.UpdateUser)
			admin.DELETE("/users/:id", adminHandler.DeleteUser)
			admin.POST("/categories", categoryHandler.CreateCategory)
			admin.DELETE("/categories/:id", categoryHandler.DeleteCategory)
		}

		api.GET("/categories", categoryHandler.GetAllCategories) // Public or Protected? Making it protected as per grouping

		api.POST("/threads", threadHandler.CreateThread)
		api.GET("/threads", threadHandler.GetAllThreads)
		api.PUT("/threads/:id", threadHandler.UpdateThread)
		api.DELETE("/threads/:id", threadHandler.DeleteThread)

		api.POST("/threads/:thread_id/posts", postHandler.CreatePost)
		api.GET("/threads/:thread_id/posts", postHandler.GetPostsByThreadID)
		api.PUT("/posts/:id", postHandler.UpdatePost)
		api.DELETE("/posts/:id", postHandler.DeletePost)

		profile := api.Group("/profile")
		{
			api.GET("/:username", profileHandler.GetProfileByUsername)
			profile.GET("/me", profileHandler.GetCurrentProfile)
			profile.PUT("", profileHandler.UpdateProfile)
		}

		api.POST("/upload", attachmentHandler.UploadAttachment)
	}

	// Start Orphan Cleanup Job (Background)
	go func() {
		// Run every 12 hours
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			log.Println("🧹 Running orphan attachment cleanup...")
			if err := attachmentService.CleanupOrphanAttachments(context.Background()); err != nil {
				log.Printf("❌ Error cleaning up orphan attachments: %v", err)
			} else {
				log.Println("✅ Orphan attachment cleanup completed.")
			}
		}
	}()

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
		&model.Category{},
		&model.Thread{},
		&model.Post{},
		&model.Attachment{},
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
