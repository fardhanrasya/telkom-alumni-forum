package server

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"anoa.com/telkomalumiforum/internal/agent"
	"anoa.com/telkomalumiforum/internal/middleware"
	"anoa.com/telkomalumiforum/pkg/storage"

	adminHttp "anoa.com/telkomalumiforum/internal/modules/admin/delivery/http"
	adminService "anoa.com/telkomalumiforum/internal/modules/admin/service"

	attachmentHttp "anoa.com/telkomalumiforum/internal/modules/attachment/delivery/http"
	attachmentRepo "anoa.com/telkomalumiforum/internal/modules/attachment/repository"
	attachmentService "anoa.com/telkomalumiforum/internal/modules/attachment/service"

	categoryHttp "anoa.com/telkomalumiforum/internal/modules/category/delivery/http"
	categoryRepo "anoa.com/telkomalumiforum/internal/modules/category/repository"
	categoryService "anoa.com/telkomalumiforum/internal/modules/category/service"

	leaderboardHttp "anoa.com/telkomalumiforum/internal/modules/leaderboard/delivery/http"
	leaderboardRepo "anoa.com/telkomalumiforum/internal/modules/leaderboard/repository"
	leaderboardService "anoa.com/telkomalumiforum/internal/modules/leaderboard/service"

	menfessHttp "anoa.com/telkomalumiforum/internal/modules/menfess/delivery/http"
	menfessRepo "anoa.com/telkomalumiforum/internal/modules/menfess/repository"
	menfessService "anoa.com/telkomalumiforum/internal/modules/menfess/service"

	notiHttp "anoa.com/telkomalumiforum/internal/modules/notification/delivery/http"
	notifRepo "anoa.com/telkomalumiforum/internal/modules/notification/repository"
	notifService "anoa.com/telkomalumiforum/internal/modules/notification/service"

	postHttp "anoa.com/telkomalumiforum/internal/modules/post/delivery/http"
	postRepo "anoa.com/telkomalumiforum/internal/modules/post/repository"
	postService "anoa.com/telkomalumiforum/internal/modules/post/service"

	profileHttp "anoa.com/telkomalumiforum/internal/modules/profile/delivery/http"
	profileService "anoa.com/telkomalumiforum/internal/modules/profile/service"

	reactionHttp "anoa.com/telkomalumiforum/internal/modules/reaction/delivery/http"
	reactionRepo "anoa.com/telkomalumiforum/internal/modules/reaction/repository"
	reactionService "anoa.com/telkomalumiforum/internal/modules/reaction/service"

	searchService "anoa.com/telkomalumiforum/internal/modules/search/service"

	statHttp "anoa.com/telkomalumiforum/internal/modules/stat/delivery/http"
	statService "anoa.com/telkomalumiforum/internal/modules/stat/service"

	threadHttp "anoa.com/telkomalumiforum/internal/modules/thread/delivery/http"
	threadRepo "anoa.com/telkomalumiforum/internal/modules/thread/repository"
	threadService "anoa.com/telkomalumiforum/internal/modules/thread/service"

	userHttp "anoa.com/telkomalumiforum/internal/modules/user/delivery/http"
	userRepo "anoa.com/telkomalumiforum/internal/modules/user/repository"
	userService "anoa.com/telkomalumiforum/internal/modules/user/service"

	viewService "anoa.com/telkomalumiforum/internal/modules/view/service"

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
	userRepo := userRepo.NewUserRepository(db)
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
	meiliSvc := searchService.NewMeiliSearchService(meiliClient)

	authSvc := userService.NewAuthService(userRepo, imageStorage, meiliSvc)
	authHandler := userHttp.NewAuthHandler(authSvc)

	adminSvc := adminService.NewAdminService(userRepo, imageStorage)
	adminHandler := adminHttp.NewAdminHandler(adminSvc)

	leaderboardRepo := leaderboardRepo.NewLeaderboardRepository(db)

	profileSvc := profileService.NewProfileService(userRepo, imageStorage, leaderboardRepo)
	profileHandler := profileHttp.NewProfileHandler(profileSvc)

	categoryRepo := categoryRepo.NewCategoryRepository(db)
	categorySvc := categoryService.NewCategoryService(categoryRepo)
	categoryHandler := categoryHttp.NewCategoryHandler(categorySvc)

	attachmentRepo := attachmentRepo.NewAttachmentRepository(db)
	attachmentSvc := attachmentService.NewAttachmentService(attachmentRepo, imageStorage)
	attachmentHandler := attachmentHttp.NewAttachmentHandler(attachmentSvc)

	// Notification Module
	notificationRepository := notifRepo.NewNotificationRepository(db)
	notificationSvc := notifService.NewNotificationService(notificationRepository, redisClient)
	notificationHandler := notiHttp.NewNotificationHandler(notificationSvc, redisClient)

	threadRepo := threadRepo.NewRepository(db)
	postRepo := postRepo.NewPostRepository(db)

	leaderboardSvc := leaderboardService.NewLeaderboardService(leaderboardRepo, userRepo, notificationSvc)
	leaderboardHandler := leaderboardHttp.NewLeaderboardHandler(leaderboardSvc)

	reactionRepo := reactionRepo.NewReactionRepository(db)
	reactionSvc := reactionService.NewReactionService(reactionRepo, redisClient, leaderboardSvc, notificationSvc, threadRepo, postRepo)
	reactionHandler := reactionHttp.NewReactionHandler(reactionSvc)

	threadSvc := threadService.NewService(threadRepo, categoryRepo, userRepo, attachmentRepo, reactionSvc, imageStorage, redisClient, meiliSvc, leaderboardSvc)
	threadHandler := threadHttp.NewThreadHandler(threadSvc)

	viewSvc := viewService.NewViewService(redisClient, threadRepo)
	if redisClient != nil {
		go viewSvc.StartViewSyncWorker(context.Background())
	}

	postSvc := postService.NewPostService(postRepo, threadRepo, userRepo, attachmentRepo, reactionSvc, imageStorage, redisClient, notificationSvc, meiliSvc, leaderboardSvc)
	postHandler := postHttp.NewPostHandler(postSvc)

	statSvc := statService.NewStatService(userRepo)
	statHandler := statHttp.NewStatHandler(statSvc, threadSvc)

	// Menfess Module
	menfessRepository := menfessRepo.NewMenfessRepository(db)
	menfessSvc := menfessService.NewMenfessService(menfessRepository, reactionSvc, redisClient)
	menfessHandler := menfessHttp.NewMenfessHandler(menfessSvc, userRepo)

	// Start AI Agent
	if redisClient != nil {
		aiAgent := agent.NewAgent(threadSvc, userRepo, categoryRepo, redisClient)
		aiAgent.Start()
	}

	// Start Orphan Cleanup Job (Background)
	go func() {
		// Run every 12 hours
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			log.Println("🧹 Running orphan attachment cleanup...")
			if err := attachmentSvc.CleanupOrphanAttachments(context.Background()); err != nil {
				log.Printf("❌ Error cleaning up orphan attachments: %v", err)
			} else {
				log.Println("✅ Orphan attachment cleanup completed.")
			}
		}
	}()

	router := gin.New()

	setupCORS(router)

	router.Use(gin.Recovery())
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/api/menfess"},
	}))

	authMiddleware := middleware.NewAuthMiddleware(userRepo)

	api := router.Group("/api")
	
	// Public routes (no auth required)
	auth := api.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.GET("/google/login", authHandler.GoogleLogin)
		auth.GET("/google/callback", authHandler.GoogleCallback)
	}

	// Protected routes (apply auth middleware explicitly)
	protected := api.Group("")
	protected.Use(authMiddleware.RequireAuth())
	{
		// Admin routes
		adminGroup := protected.Group("/admin")
		adminGroup.Use(authMiddleware.RequireAdmin())
		{
			adminGroup.POST("/users", adminHandler.CreateUser)
			adminGroup.GET("/users", adminHandler.GetAllUsers)
			adminGroup.PUT("/users/:id", adminHandler.UpdateUser)
			adminGroup.DELETE("/users/:id", adminHandler.DeleteUser)
			adminGroup.POST("/categories", categoryHandler.CreateCategory)
			adminGroup.DELETE("/categories/:id", categoryHandler.DeleteCategory)
		}

		// User routes
		protected.GET("/users/count", statHandler.GetTotalUsers)
		protected.GET("/categories", categoryHandler.GetAllCategories)
		
		// Thread routes
		protected.POST("/threads", threadHandler.CreateThread)
		protected.GET("/threads", threadHandler.GetAllThreads)
		protected.GET("/threads/me", threadHandler.GetMyThreads)
		protected.GET("/threads/trending", statHandler.GetTrendingThreads)
		protected.GET("/threads/user/:username", threadHandler.GetThreadsByUsername)
		protected.GET("/threads/slug/:slug", threadHandler.GetThreadBySlug)
		protected.PUT("/threads/:thread_id", threadHandler.UpdateThread)
		protected.DELETE("/threads/:thread_id", threadHandler.DeleteThread)
		protected.POST("/threads/:thread_id/posts", postHandler.CreatePost)
		protected.GET("/threads/:thread_id/posts", postHandler.GetPostsByThreadID)

		// Post routes
		protected.GET("/posts/:post_id", postHandler.GetPostByID)
		protected.PUT("/posts/:post_id", postHandler.UpdatePost)
		protected.DELETE("/posts/:post_id", postHandler.DeletePost)

		// Profile routes
		protected.GET("/profile/:username", profileHandler.GetProfileByUsername)
		protected.GET("/profile/me", profileHandler.GetCurrentProfile)
		protected.PUT("/profile", profileHandler.UpdateProfile)
		
		// Notification routes
		protected.GET("/notifications", notificationHandler.GetNotifications)
		protected.GET("/notifications/unread-count", notificationHandler.UnreadCount)
		protected.PUT("/notifications/:id/read", notificationHandler.MarkAsRead)
		protected.PUT("/notifications/read-all", notificationHandler.MarkAllAsRead)
		protected.GET("/notifications/ws", notificationHandler.HandleWebSocket)
		
		// Menfess routes
		protected.POST("/menfess", menfessHandler.CreateMenfess)
		protected.GET("/menfess", menfessHandler.GetMenfesses)

		// Reaction routes
		protected.POST("/reactions", reactionHandler.ToggleReaction)
		protected.GET("/reactions/:refType/:refID", reactionHandler.GetReactions)
		
		// Other protected routes
		protected.POST("/upload", attachmentHandler.UploadAttachment)
		protected.GET("/leaderboard", leaderboardHandler.GetLeaderboard)
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
