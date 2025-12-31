package server

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"anoa.com/telkomalumiforum/internal/agent"
	"anoa.com/telkomalumiforum/internal/handler"
	"anoa.com/telkomalumiforum/internal/middleware"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/internal/service"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/meilisearch/meilisearch-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Server struct {
	engine      *gin.Engine
	db          *gorm.DB
	redisClient *redis.Client
}

func NewServer(db *gorm.DB, redisClient *redis.Client) *Server {
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
	if !strings.HasPrefix(meiliHost, "http") {
		meiliHost = "http://" + meiliHost + ":7700"
	}

	meiliClient := meilisearch.New(meiliHost, meilisearch.WithAPIKey(os.Getenv("MEILI_MASTER_KEY")))
	meiliService := service.NewMeiliSearchService(meiliClient)

	authService := service.NewAuthService(userRepo, imageStorage, meiliService)
	authHandler := handler.NewAuthHandler(authService)

	adminService := service.NewAdminService(userRepo, imageStorage)
	adminHandler := handler.NewAdminHandler(adminService)

	leaderboardRepo := repository.NewLeaderboardRepository(db)

	profileService := service.NewProfileService(userRepo, imageStorage, leaderboardRepo)
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

	leaderboardService := service.NewLeaderboardService(leaderboardRepo, userRepo, notificationService)
	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardService)

	reactionRepo := repository.NewReactionRepository(db)
	reactionService := service.NewReactionService(reactionRepo, redisClient, leaderboardService, notificationService, threadRepo, postRepo)
	reactionHandler := handler.NewReactionHandler(reactionService)

	threadService := service.NewThreadService(threadRepo, categoryRepo, userRepo, attachmentRepo, reactionService, imageStorage, redisClient, meiliService, leaderboardService)
	threadHandler := handler.NewThreadHandler(threadService)

	viewService := service.NewViewService(redisClient, threadRepo)
	if redisClient != nil {
		go viewService.StartViewSyncWorker(context.Background())
	}

	postService := service.NewPostService(postRepo, threadRepo, userRepo, attachmentRepo, reactionService, imageStorage, redisClient, notificationService, meiliService, leaderboardService)
	postHandler := handler.NewPostHandler(postService)

	statService := service.NewStatService(userRepo)
	statHandler := handler.NewStatHandler(statService, threadService)

	menfessRepo := repository.NewMenfessRepository(db)
	menfessService := service.NewMenfessService(menfessRepo, reactionService, redisClient)
	menfessHandler := handler.NewMenfessHandler(menfessService, userRepo)

	// Start AI Agent
	if redisClient != nil {
		aiAgent := agent.NewAgent(threadService, userRepo, categoryRepo, redisClient)
		aiAgent.Start()
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

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/api/menfess"},
	}))

	setupCORS(router)

	authMiddleware := middleware.NewAuthMiddleware(userRepo)

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		auth.POST("/login", authHandler.Login)
		auth.GET("/google/login", authHandler.GoogleLogin)
		auth.GET("/google/callback", authHandler.GoogleCallback)
	}

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
		api.GET("/categories", categoryHandler.GetAllCategories)
		
		threads := api.Group("/threads") 
		{
			threads.POST("/", threadHandler.CreateThread)
			threads.GET("/", threadHandler.GetAllThreads)
			threads.GET("/me", threadHandler.GetMyThreads)
			threads.GET("/user/:username", threadHandler.GetThreadsByUsername)
			threads.GET("/slug/:slug", threadHandler.GetThreadBySlug)
			threads.PUT("/:thread_id", threadHandler.UpdateThread)
			threads.DELETE("/:thread_id", threadHandler.DeleteThread)
			threads.POST("/:thread_id/posts", postHandler.CreatePost)
			threads.GET("/:thread_id/posts", postHandler.GetPostsByThreadID)
			threads.GET("/trending", statHandler.GetTrendingThreads)
		}

		posts := api.Group("/posts") 
		{
			posts.GET("/:post_id", postHandler.GetPostByID)
			posts.PUT("/:post_id", postHandler.UpdatePost)
			posts.DELETE("/:post_id", postHandler.DeletePost)
		}

		profile := api.Group("/profile")
		{
			profile.GET("/:username", profileHandler.GetProfileByUsername)
			profile.GET("/me", profileHandler.GetCurrentProfile)
			profile.PUT("", profileHandler.UpdateProfile)
		}
		
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

		reactions := api.Group("/reactions") 
		{
			reactions.POST("/", reactionHandler.ToggleReaction)
			reactions.GET("/:refType/:refID", reactionHandler.GetReactions)
		}
		
		api.POST("/upload", attachmentHandler.UploadAttachment)
		api.GET("/leaderboard", leaderboardHandler.GetLeaderboard)
	}

	return &Server{
		engine:      router,
		db:          db,
		redisClient: redisClient,
	}
}

func (s *Server) Run(addr string) error {
	return s.engine.Run(addr)
}

func setupCORS(router *gin.Engine) {
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	var origins []string
	if allowedOrigins != "" {
		origins = strings.Split(allowedOrigins, ",")
	} else {
		origins = []string{"http://localhost:3000"}
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
}
