package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"anoa.com/telkomalumiforum/internal/dto"
	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type ThreadService interface {
	CreateThread(ctx context.Context, userID uuid.UUID, req dto.CreateThreadRequest) error
	GetAllThreads(ctx context.Context, userID uuid.UUID, filter dto.ThreadFilter) (*dto.PaginatedThreadResponse, error)
	GetMyThreads(ctx context.Context, userID uuid.UUID, page, limit int) (*dto.PaginatedThreadResponse, error)
	GetThreadBySlug(ctx context.Context, slug string) (*dto.ThreadResponse, error)
	DeleteThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error
	UpdateThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID, req dto.UpdateThreadRequest) error
	IncrementView(ctx context.Context, threadID uuid.UUID, userID uuid.UUID) error
	GetThreadsByUsername(ctx context.Context, currentUserID uuid.UUID, username string, page, limit int) (*dto.PaginatedThreadResponse, error)
	GetTrendingThreads(ctx context.Context, limit int) ([]dto.ThreadResponse, error)
}

type threadService struct {
	threadRepo         repository.ThreadRepository
	categoryRepo       repository.CategoryRepository
	userRepo           repository.UserRepository
	attachmentRepo     repository.AttachmentRepository
	reactionService    ReactionService
	fileStorage        storage.ImageStorage
	redisClient        *redis.Client
	viewService        ViewService
	meili              MeiliSearchService
	leaderboardService LeaderboardService
}

func NewThreadService(threadRepo repository.ThreadRepository, categoryRepo repository.CategoryRepository, userRepo repository.UserRepository, attachmentRepo repository.AttachmentRepository, reactionService ReactionService, fileStorage storage.ImageStorage, redisClient *redis.Client, meili MeiliSearchService, leaderboardService LeaderboardService) ThreadService {
	viewService := NewViewService(redisClient, threadRepo)

	return &threadService{
		threadRepo:         threadRepo,
		categoryRepo:       categoryRepo,
		userRepo:           userRepo,
		attachmentRepo:     attachmentRepo,
		reactionService:    reactionService,
		fileStorage:        fileStorage,
		redisClient:        redisClient,
		viewService:        viewService,
		meili:              meili,
		leaderboardService: leaderboardService,
	}
}

// ... (CreateThread remains unchanged)




func (s *threadService) CreateThread(ctx context.Context, userID uuid.UUID, req dto.CreateThreadRequest) error {
	// Rate Limiting
	// 1. Global Cooldown
	globalLimit := GetDurationFromEnv("RATE_LIMIT_GLOBAL", 5*time.Second)
	allowed, err := CheckAndSetRateLimit(ctx, s.redisClient, userID, "global", globalLimit)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !allowed {
		ttl, _ := GetRateLimitTTL(ctx, s.redisClient, userID, "global")
		return &RateLimitError{
			Message:    fmt.Sprintf("you are doing that too fast. Please wait %.0f seconds", ttl.Seconds()),
			RetryAfter: ttl,
		}
	}

	// 2. Thread-specific Cooldown
	threadLimit := GetDurationFromEnv("RATE_LIMIT_THREAD", 5*time.Minute)
	allowed, err = CheckAndSetRateLimit(ctx, s.redisClient, userID, "thread", threadLimit)
	if err != nil {
		_ = ClearRateLimit(ctx, s.redisClient, userID, "global") // Rollback global
		return fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !allowed {
		_ = ClearRateLimit(ctx, s.redisClient, userID, "global") // Rollback global
		ttl, _ := GetRateLimitTTL(ctx, s.redisClient, userID, "thread")
		return &RateLimitError{
			Message:    fmt.Sprintf("you can only create one thread every %.0f minutes. Please wait %.0f minutes", threadLimit.Minutes(), ttl.Minutes()),
			RetryAfter: ttl,
		}
	}

	// Defer rollback in case of creation failure
	creationFailed := true
	defer func() {
		if creationFailed {
			_ = ClearRateLimit(ctx, s.redisClient, userID, "global")
			_ = ClearRateLimit(ctx, s.redisClient, userID, "thread")
		}
	}()

	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Validate Audience based on Role
	switch user.Role.Name {
	case "siswa":
		if req.Audience == "guru" {
			return fmt.Errorf("siswa cannot create thread for guru")
		}
	case "guru":
		if req.Audience == "siswa" {
			return fmt.Errorf("guru cannot create thread for siswa")
		}
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return fmt.Errorf("invalid category id format")
	}

	category, err := s.categoryRepo.FindByID(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("invalid category id")
	}

	// Sanitize slug
	slug := strings.ToLower(req.Title)
	// Remove invalid chars
	reg, _ := regexp.Compile("[^a-z0-9 ]+")
	slug = reg.ReplaceAllString(slug, "")
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	// Trim hyphens
	slug = strings.Trim(slug, "-")

	// Basic slug uniqueness check
	existing, _ := s.threadRepo.FindBySlug(ctx, slug)
	if existing != nil {
		// Append a short random string or timestamp for uniqueness
		slug = fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])
	}

	thread := &model.Thread{
		CategoryID: &category.ID,
		UserID:     userID,
		Title:      req.Title,
		Slug:       slug,
		Content:    req.Content,
		Audience:   req.Audience,
	}

	if err := s.threadRepo.Create(ctx, thread); err != nil {
		return err
	}

	if len(req.AttachmentIDs) > 0 {
		if err := s.attachmentRepo.UpdateThreadID(ctx, req.AttachmentIDs, thread.ID, userID); err != nil {
			return err
		}
	}

	// Everything succeeded, don't roll back the rate limits.
	creationFailed = false

	// Index to Meilisearch
	thread.User = *user
	if s.meili != nil {
		if err := s.meili.IndexThread(thread); err != nil {
			// Log error but don't fail the request?
			// Or fail? Best to log.
			fmt.Printf("Failed to index thread: %v\n", err)
		}
	}

	// Add Gamification Points (nil actorID for self-actions like create_thread)
	if s.leaderboardService != nil {
		s.leaderboardService.AddGamificationPointsAsync(thread.UserID, ActionCreateThread, thread.ID.String(), "threads", nil)
	}

	return nil
}

func (s *threadService) GetAllThreads(ctx context.Context, userID uuid.UUID, filter dto.ThreadFilter) (*dto.PaginatedThreadResponse, error) {
	// 1. Fetch User Role
	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// 2. Determine Allowed Audiences
	var effectiveAudiences []string
	var allowed []string

	switch user.Role.Name {
	case "siswa":
		allowed = []string{"siswa", "semua"}
	case "guru":
		allowed = []string{"guru", "semua"}
	default:
		// Admin sees everything, leave allowed empty to indicate "all" or specific logic
		// But Wait, if filter is set, we use filter. If not set, we return all.
		// Existing logic: if len(audiences) > 0 -> WHERE audience IN...
		// So for admin, if we keep effectiveAudiences empty, it means no WHERE audience constraint -> ALL.
	}

	if len(allowed) > 0 {
		if filter.Audience != "" {
			// Check if requested audience is allowed
			isAllowed := false
			for _, a := range allowed {
				if a == filter.Audience {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				return &dto.PaginatedThreadResponse{
					Data: []dto.ThreadResponse{},
					Meta: dto.PaginationMeta{
						CurrentPage: filter.Page,
						TotalPages:  0,
						TotalItems:  0,
						Limit:       filter.Limit,
					},
				}, nil
			}
			effectiveAudiences = []string{filter.Audience}
		} else {
			effectiveAudiences = allowed
		}
	} else {
		// Admin logic
		if filter.Audience != "" {
			effectiveAudiences = []string{filter.Audience}
		}
	}

	var categoryID *uuid.UUID
	if filter.CategoryID != "" {
		id, err := uuid.Parse(filter.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("invalid category id")
		}
		categoryID = &id
	}

	offset := (filter.Page - 1) * filter.Limit
	threads, total, err := s.threadRepo.FindAll(ctx, categoryID, filter.Search, effectiveAudiences, filter.SortBy, offset, filter.Limit)
	if err != nil {
		return nil, err
	}

	var threadResponses []dto.ThreadResponse
	for _, thread := range threads {
		var attachments []dto.AttachmentResponse
		for _, att := range thread.Attachments {
			attachments = append(attachments, dto.AttachmentResponse{
				ID:       att.ID,
				FileURL:  att.FileURL,
				FileType: att.FileType,
			})
		}

		authorResponse := dto.AuthorResponse{
			Username: "Unknown",
		}
		if thread.User.Username != "" {
			authorResponse.Username = thread.User.Username
			authorResponse.AvatarURL = thread.User.AvatarURL
		}

		reactions, _ := s.reactionService.GetReactions(ctx, &userID, thread.ID, "thread")
		likesCount := int64(reactions.Counts["👍"])

		resp := dto.ThreadResponse{
			ID:           thread.ID,
			CategoryName: thread.Category.Name,
			Title:        thread.Title,
			Slug:         thread.Slug,
			Content:      thread.Content,
			Audience:     thread.Audience,
			Views:        thread.Views,
			Author:       authorResponse,
			Attachments:  attachments,
			LikesCount:   likesCount,
			Reactions:    *reactions,
			CreatedAt:    thread.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		threadResponses = append(threadResponses, resp)
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit != 0 {
		totalPages++
	}

	return &dto.PaginatedThreadResponse{
		Data: threadResponses,
		Meta: dto.PaginationMeta{
			CurrentPage: filter.Page,
			TotalPages:  totalPages,
			TotalItems:  total,
			Limit:       filter.Limit,
		},
	}, nil
}

func (s *threadService) GetMyThreads(ctx context.Context, userID uuid.UUID, page, limit int) (*dto.PaginatedThreadResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	offset := (page - 1) * limit
	threads, total, err := s.threadRepo.FindByUserID(ctx, userID, nil, offset, limit)
	if err != nil {
		return nil, err
	}

	var threadResponses []dto.ThreadResponse
	for _, thread := range threads {
		var attachments []dto.AttachmentResponse
		for _, att := range thread.Attachments {
			attachments = append(attachments, dto.AttachmentResponse{
				ID:       att.ID,
				FileURL:  att.FileURL,
				FileType: att.FileType,
			})
		}

		authorResponse := dto.AuthorResponse{
			Username: "Unknown",
		}
		if thread.User.Username != "" {
			authorResponse.Username = thread.User.Username
			authorResponse.AvatarURL = thread.User.AvatarURL
		}

		reactions, _ := s.reactionService.GetReactions(ctx, &userID, thread.ID, "thread")
		likesCount := int64(reactions.Counts["👍"])

		resp := dto.ThreadResponse{
			ID:           thread.ID,
			CategoryName: thread.Category.Name,
			Title:        thread.Title,
			Slug:         thread.Slug,
			Content:      thread.Content,
			Audience:     thread.Audience,
			Views:        thread.Views,
			Author:       authorResponse,
			Attachments:  attachments,
			LikesCount:   likesCount,
			Reactions:    *reactions,
			CreatedAt:    thread.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		threadResponses = append(threadResponses, resp)
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &dto.PaginatedThreadResponse{
		Data: threadResponses,
		Meta: dto.PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  total,
			Limit:       limit,
		},
	}, nil
}

func (s *threadService) GetThreadsByUsername(ctx context.Context, currentUserID uuid.UUID, username string, page, limit int) (*dto.PaginatedThreadResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	currentUser, err := s.userRepo.FindByID(ctx, currentUserID.String())
	if err != nil {
		return nil, fmt.Errorf("current user not found")
	}

	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	var allowedAudiences []string
	switch currentUser.Role.Name {
	case "siswa":
		allowedAudiences = []string{"semua", "siswa"}
	case "guru":
		allowedAudiences = []string{"semua", "guru"}
	case "admin":
		allowedAudiences = nil // See all
	default:
		allowedAudiences = []string{"semua"}
	}

	offset := (page - 1) * limit
	threads, total, err := s.threadRepo.FindByUserID(ctx, user.ID, allowedAudiences, offset, limit)
	if err != nil {
		return nil, err
	}

	var threadResponses []dto.ThreadResponse
	for _, thread := range threads {
		var attachments []dto.AttachmentResponse
		for _, att := range thread.Attachments {
			attachments = append(attachments, dto.AttachmentResponse{
				ID:       att.ID,
				FileURL:  att.FileURL,
				FileType: att.FileType,
			})
		}

		authorResponse := dto.AuthorResponse{
			Username: "Unknown",
		}
		if thread.User.Username != "" {
			authorResponse.Username = thread.User.Username
			authorResponse.AvatarURL = thread.User.AvatarURL
		}

		reactions, _ := s.reactionService.GetReactions(ctx, &currentUserID, thread.ID, "thread")
		likesCount := int64(reactions.Counts["👍"])

		resp := dto.ThreadResponse{
			ID:           thread.ID,
			CategoryName: thread.Category.Name,
			Title:        thread.Title,
			Slug:         thread.Slug,
			Content:      thread.Content,
			Audience:     thread.Audience,
			Views:        thread.Views,
			Author:       authorResponse,
			Attachments:  attachments,
			LikesCount:   likesCount,
			Reactions:    *reactions,
			CreatedAt:    thread.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		threadResponses = append(threadResponses, resp)
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &dto.PaginatedThreadResponse{
		Data: threadResponses,
		Meta: dto.PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  total,
			Limit:       limit,
		},
	}, nil
}

func (s *threadService) GetThreadBySlug(ctx context.Context, slug string) (*dto.ThreadResponse, error) {
	thread, err := s.threadRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	var attachments []dto.AttachmentResponse
	for _, att := range thread.Attachments {
		attachments = append(attachments, dto.AttachmentResponse{
			ID:       att.ID,
			FileURL:  att.FileURL,
			FileType: att.FileType,
		})
	}

	authorResponse := dto.AuthorResponse{
		Username: "Unknown",
	}
	if thread.User.Username != "" {
		authorResponse.Username = thread.User.Username
		authorResponse.AvatarURL = thread.User.AvatarURL
	}

	reactions, _ := s.reactionService.GetReactions(ctx, nil, thread.ID, "thread")
	likesCount := int64(reactions.Counts["👍"])

	return &dto.ThreadResponse{
		ID:           thread.ID,
		CategoryName: thread.Category.Name,
		Title:        thread.Title,
		Slug:         thread.Slug,
		Content:      thread.Content,
		Audience:     thread.Audience,
		Views:        thread.Views,
		Author:       authorResponse,
		Attachments:  attachments,
		LikesCount:   likesCount,
		Reactions:    *reactions,
		CreatedAt:    thread.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *threadService) IncrementView(ctx context.Context, threadID uuid.UUID, userID uuid.UUID) error {
	return s.viewService.IncrementView(ctx, threadID, userID)
}

func (s *threadService) DeleteThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error {
	// 1. Get Thread
	thread, err := s.threadRepo.FindByID(ctx, threadID)
	if err != nil {
		return err
	}

	// 2. Get Requesting User to check Role
	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// 3. Permission Check
	// Assuming "admin" is the role name for admin users.
	if thread.UserID != userID && user.Role.Name != "admin" {
		return fmt.Errorf("unauthorized: you can only delete your own threads unless you are an admin")
	}

	// 4. Delete Attachments
	for _, att := range thread.Attachments {
		// Delete from Cloudinary
		// We ignore error here to proceed with deletion, but ideally log it.
		// Since we don't have a logger injected, we'll just proceed.
		_ = s.fileStorage.DeleteImage(ctx, att.FileURL)

		// Delete from DB
		if err := s.attachmentRepo.Delete(ctx, att.ID); err != nil {
			return fmt.Errorf("failed to delete attachment record: %w", err)
		}
	}

	// 5. Delete Thread
	if err := s.threadRepo.Delete(ctx, threadID); err != nil {
		return err
	}

	if s.meili != nil {
		_ = s.meili.DeleteThread(threadID.String())
	}

	return nil
}

func (s *threadService) UpdateThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID, req dto.UpdateThreadRequest) error {
	thread, err := s.threadRepo.FindByID(ctx, threadID)
	if err != nil {
		return err
	}

	if thread.UserID != userID {
		return fmt.Errorf("unauthorized: you can only update your own thread")
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return fmt.Errorf("invalid category id format")
	}

	thread.Title = req.Title
	thread.Content = req.Content
	thread.CategoryID = &categoryID

	// Validate Audience based on Role
	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("user not found")
	}

	switch user.Role.Name {
	case "siswa":
		if req.Audience == "guru" {
			return fmt.Errorf("siswa cannot set audience to guru")
		}
	case "guru":
		if req.Audience == "siswa" {
			return fmt.Errorf("guru cannot set audience to siswa")
		}
	}
	thread.Audience = req.Audience

	// Update Attachments
	// 1. Identify which attachments to keep vs delete
	currentAttachments := make(map[uint]model.Attachment)
	for _, att := range thread.Attachments {
		currentAttachments[att.ID] = att
	}

	desiredAttachments := make(map[uint]bool)
	for _, id := range req.AttachmentIDs {
		desiredAttachments[id] = true
	}

	// Delete removed attachments
	for id, att := range currentAttachments {
		if !desiredAttachments[id] {
			_ = s.fileStorage.DeleteImage(ctx, att.FileURL)
			_ = s.attachmentRepo.Delete(ctx, id)
		}
	}

	// Add new attachments (orphan or just ensure link)
	if len(req.AttachmentIDs) > 0 {
		if err := s.attachmentRepo.UpdateThreadID(ctx, req.AttachmentIDs, thread.ID, userID); err != nil {
			return err
		}
	}

	if err := s.threadRepo.Update(ctx, thread); err != nil {
		return err
	}

	if s.meili != nil {
		// Reload thread to get fresh associations for indexing
		reloadedThread, err := s.threadRepo.FindByID(ctx, threadID)
		if err == nil {
			_ = s.meili.IndexThread(reloadedThread)
		}
	}

	return nil
}
