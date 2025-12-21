package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"anoa.com/telkomalumiforum/internal/handler"
	"anoa.com/telkomalumiforum/internal/middleware"
	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/internal/service"
	"anoa.com/telkomalumiforum/pkg/database"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/meilisearch/meilisearch-go"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	db := database.Connect()

	// Initialize Redis
	redisClient, err := database.ConnectRedis()
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis, like feature will not work: %v", err)
	}

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

	// Initialize Meilisearch
	meiliHost := os.Getenv("MEILISEARCH_HOST")
	if meiliHost == "" {
		meiliHost = "http://localhost:7700"
	}
	// Basic check to ensure protocol presence if needed, but assuming env is correct or handled
	if !strings.HasPrefix(meiliHost, "http") {
		// If provided as "meilisearch", likely from docker, assuming port 7700
		meiliHost = "http://" + meiliHost + ":7700"
	}

	meiliClient := meilisearch.New(meiliHost, meilisearch.WithAPIKey(os.Getenv("MEILI_MASTER_KEY")))

	meiliService := service.NewMeiliSearchService(meiliClient)

	authService := service.NewAuthService(userRepo, imageStorage, meiliService)
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

	notificationRepo := repository.NewNotificationRepository(db)
	notificationService := service.NewNotificationService(notificationRepo, redisClient)
	notificationHandler := handler.NewNotificationHandler(notificationService, redisClient)

	threadRepo := repository.NewThreadRepository(db)
	postRepo := repository.NewPostRepository(db)

	likeRepo := repository.NewLikeRepository(db)
	likeService := service.NewLikeService(redisClient, likeRepo, threadRepo, postRepo, notificationService)
	likeHandler := handler.NewLikeHandler(likeService)

	threadService := service.NewThreadService(threadRepo, categoryRepo, userRepo, attachmentRepo, likeService, imageStorage, redisClient, meiliService)
	threadHandler := handler.NewThreadHandler(threadService)

	viewService := service.NewViewService(redisClient, threadRepo)
	if redisClient != nil {
		go viewService.StartViewSyncWorker(context.Background())
	}

	postService := service.NewPostService(postRepo, threadRepo, userRepo, attachmentRepo, likeService, imageStorage, redisClient, notificationService, meiliService)
	postHandler := handler.NewPostHandler(postService)

	// Start Like Worker
	if redisClient != nil {
		go likeService.StartWorker(context.Background())
	}

	statService := service.NewStatService(userRepo)
	statHandler := handler.NewStatHandler(statService, threadService)

	menfessRepo := repository.NewMenfessRepository(db)
	menfessService := service.NewMenfessService(menfessRepo, redisClient)
	menfessHandler := handler.NewMenfessHandler(menfessService, userRepo)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/api/menfess"},
	}))

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	var origins []string
	if allowedOrigins != "" {
		origins = strings.Split(allowedOrigins, ",")
	} else {
		origins = []string{"http://localhost:3000"}
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins: origins,

		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},

		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},

		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		auth.POST("/login", authHandler.Login)
	}

	authMiddleware := middleware.NewAuthMiddleware(userRepo)

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

		api.GET("/users/count", statHandler.GetTotalUsers)
		api.GET("/threads/trending", statHandler.GetTrendingThreads)

		api.GET("/categories", categoryHandler.GetAllCategories)

		api.POST("/threads", threadHandler.CreateThread)
		api.GET("/threads", threadHandler.GetAllThreads)
		api.GET("/threads/me", threadHandler.GetMyThreads)
		api.GET("/threads/user/:username", threadHandler.GetThreadsByUsername)
		api.GET("/threads/slug/:slug", threadHandler.GetThreadBySlug)
		api.PUT("/threads/:thread_id", threadHandler.UpdateThread)
		api.DELETE("/threads/:thread_id", threadHandler.DeleteThread)

		api.POST("/threads/:thread_id/posts", postHandler.CreatePost)
		api.GET("/threads/:thread_id/posts", postHandler.GetPostsByThreadID)
		api.GET("/posts/:post_id", postHandler.GetPostByID)
		api.PUT("/posts/:post_id", postHandler.UpdatePost)
		api.DELETE("/posts/:post_id", postHandler.DeletePost)

		api.POST("/threads/:thread_id/like", likeHandler.LikeThread)
		api.GET("/threads/:thread_id/like", likeHandler.CheckThreadLike)
		api.DELETE("/threads/:thread_id/like", likeHandler.UnlikeThread)
		api.POST("/posts/:post_id/like", likeHandler.LikePost)
		api.GET("/posts/:post_id/like", likeHandler.CheckPostLike)
		api.DELETE("/posts/:post_id/like", likeHandler.UnlikePost)

		profile := api.Group("/profile")
		{
			profile.GET("/:username", profileHandler.GetProfileByUsername)
			profile.GET("/me", profileHandler.GetCurrentProfile)
			profile.PUT("", profileHandler.UpdateProfile)
		}

		api.POST("/upload", attachmentHandler.UploadAttachment)

		notifications := api.Group("/notifications")
		{
			notifications.GET("", notificationHandler.GetNotifications)
			notifications.GET("/unread-count", notificationHandler.UnreadCount)
			notifications.PUT("/:id/read", notificationHandler.MarkAsRead)
			notifications.PUT("/read-all", notificationHandler.MarkAllAsRead)
			notifications.GET("/ws", notificationHandler.HandleWebSocket)
		}

		menfess := api.Group("/menfess")
		{
			menfess.POST("", menfessHandler.CreateMenfess)
			menfess.GET("", menfessHandler.GetMenfesses)
		}
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
		&model.ThreadLike{},
		&model.ThreadLike{},
		&model.PostLike{},
		&model.Notification{},
		&model.Menfess{},
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
