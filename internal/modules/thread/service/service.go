package thread

import (
	"context"
	"fmt"

	"anoa.com/telkomalumiforum/internal/entity"
	attachmentRepo "anoa.com/telkomalumiforum/internal/modules/attachment/repository"
	categoryRepo "anoa.com/telkomalumiforum/internal/modules/category/repository"
	threadDto "anoa.com/telkomalumiforum/internal/modules/thread/dto"
	repo "anoa.com/telkomalumiforum/internal/modules/thread/repository"
	userRepo "anoa.com/telkomalumiforum/internal/modules/user/repository"
	"anoa.com/telkomalumiforum/pkg/apperror"
	commonDto "anoa.com/telkomalumiforum/pkg/dto"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	leaderboard "anoa.com/telkomalumiforum/internal/modules/leaderboard/service"
	reaction "anoa.com/telkomalumiforum/internal/modules/reaction/service"
	search "anoa.com/telkomalumiforum/internal/modules/search/service"
	view "anoa.com/telkomalumiforum/internal/modules/view/service"
)

type Service interface {
	CreateThread(ctx context.Context, userID uuid.UUID, req threadDto.CreateThreadRequest) error
	GetAllThreads(ctx context.Context, userID uuid.UUID, filter commonDto.ThreadFilter) (*commonDto.PaginatedThreadResponse, error)
	GetMyThreads(ctx context.Context, userID uuid.UUID, page, limit int) (*commonDto.PaginatedThreadResponse, error)
	GetThreadBySlug(ctx context.Context, slug string) (*commonDto.ThreadResponse, error)
	DeleteThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error
	UpdateThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID, req threadDto.UpdateThreadRequest) error
	IncrementView(ctx context.Context, threadID uuid.UUID, userID uuid.UUID) error
	GetThreadsByUsername(ctx context.Context, currentUserID uuid.UUID, username string, page, limit int) (*commonDto.PaginatedThreadResponse, error)
	GetTrendingThreads(ctx context.Context, limit int) ([]commonDto.ThreadResponse, error)
}

type service struct {
	threadRepo         repo.Repository
	categoryRepo       categoryRepo.CategoryRepository
	userRepo           userRepo.UserRepository
	attachmentRepo     attachmentRepo.AttachmentRepository
	reactionService    reaction.ReactionService
	fileStorage        storage.ImageStorage
	redisClient        *redis.Client
	viewService        view.ViewService
	meili              search.MeiliSearchService
	leaderboardService leaderboard.LeaderboardService
}

func NewService(threadRepo repo.Repository, categoryRepo categoryRepo.CategoryRepository, userRepo userRepo.UserRepository, attachmentRepo attachmentRepo.AttachmentRepository, reactionService reaction.ReactionService, fileStorage storage.ImageStorage, redisClient *redis.Client, meili search.MeiliSearchService, leaderboardService leaderboard.LeaderboardService) Service {
	// viewService := view.NewViewService(redisClient, threadRepo)
	// ViewService likely expects repository.ThreadRepository, we need to check if we broke it.
	// The ViewService signature wasn't refactored yet. If ViewService expects the OLD interface, it will fail because we are passing the NEW interface.
	// But wait, the OLD interface `repository.ThreadRepository` checks methods. The new `repo.Repository` has SAME methods.
	// Go interfaces are implicit. So as long as `repo.Repository` has methods of `repository.ThreadRepository` (if it was defined in repository package), it matches.
	// However, `repository.ThreadRepository` was DELETED/MOVED.
	// So `view.NewViewService` signature expects WHAT? It expects `repository.ThreadRepository` which is now GONE from `internal/repository`.
	// We must Fix `view` service too.

	// For now, I will comment out ViewService usage or assume I'm fixing it.
	// To be safe, I will invoke a fix on ViewService in the next step.
	// But `view.NewViewService` is imported.
	// The `view` package imports `internal/repository` which NO LONGER has `ThreadRepository`.
	// So `view` package will fail to compile.

	viewService := view.NewViewService(redisClient, threadRepo)

	return &service{
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

func (s *service) CreateThread(ctx context.Context, userID uuid.UUID, req threadDto.CreateThreadRequest) error {
	// 1. Rate Limiting
	cleanup, err := s.checkCreateThreadRateLimit(ctx, userID)
	if err != nil {
		return err
	}
	// Defer rollback if creation fails
	creationFailed := true
	defer func() {
		if creationFailed && cleanup != nil {
			cleanup()
		}
	}()

	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("user not found: %w", apperror.ErrNotFound)
	}

	// 2. Validate Audience
	if err := s.validateAudienceForRole(user.Role.Name, req.Audience); err != nil {
		return err
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return fmt.Errorf("invalid category id format: %w", apperror.ErrBadRequest)
	}

	category, err := s.categoryRepo.FindByID(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("invalid category id: %w", apperror.ErrBadRequest)
	}

	// 3. Generate Slug
	slug := s.generateUniqueSlug(ctx, req.Title)

	thread := &entity.Thread{
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

	// 4. Handle Attachments
	if err := s.processThreadAttachments(ctx, thread.ID, userID, req.AttachmentIDs); err != nil {
		return err
	}

	// Success
	creationFailed = false

	// Index Meilisearch
	thread.User = *user
	if s.meili != nil {
		if err := s.meili.IndexThread(thread); err != nil {
			fmt.Printf("Failed to index thread: %v\n", err)
		}
	}

	// Gamification
	if s.leaderboardService != nil {
		s.leaderboardService.AddGamificationPointsAsync(thread.UserID, leaderboard.ActionCreateThread, thread.ID.String(), "threads", nil)
	}

	return nil
}

func (s *service) GetAllThreads(ctx context.Context, userID uuid.UUID, filter commonDto.ThreadFilter) (*commonDto.PaginatedThreadResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", apperror.ErrNotFound)
	}

	allowed := s.determineAllowedAudiences(user.Role.Name)
	var effectiveAudiences []string

	if len(allowed) > 0 {
		if filter.Audience != "" {
			isAllowed := false
			for _, a := range allowed {
				if a == filter.Audience {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				// Return empty if filtered audience is not allowed
				return &commonDto.PaginatedThreadResponse{
					Data: []commonDto.ThreadResponse{},
					Meta: commonDto.PaginationMeta{
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
		// Admin/All check
		if filter.Audience != "" {
			effectiveAudiences = []string{filter.Audience}
		}
	}

	var categoryID *uuid.UUID
	if filter.CategoryID != "" {
		id, err := uuid.Parse(filter.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("invalid category id: %w", apperror.ErrBadRequest)
		}
		categoryID = &id
	}

	offset := (filter.Page - 1) * filter.Limit
	threads, total, err := s.threadRepo.FindAll(ctx, categoryID, filter.Search, effectiveAudiences, filter.SortBy, offset, filter.Limit)
	if err != nil {
		return nil, err
	}

	var threadResponses []commonDto.ThreadResponse
	for _, thread := range threads {
		threadResponses = append(threadResponses, s.buildThreadResponse(ctx, *thread, &userID))
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit != 0 {
		totalPages++
	}

	return &commonDto.PaginatedThreadResponse{
		Data: threadResponses,
		Meta: commonDto.PaginationMeta{
			CurrentPage: filter.Page,
			TotalPages:  totalPages,
			TotalItems:  total,
			Limit:       filter.Limit,
		},
	}, nil
}

func (s *service) GetMyThreads(ctx context.Context, userID uuid.UUID, page, limit int) (*commonDto.PaginatedThreadResponse, error) {
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

	var threadResponses []commonDto.ThreadResponse
	for _, thread := range threads {
		threadResponses = append(threadResponses, s.buildThreadResponse(ctx, *thread, &userID))
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &commonDto.PaginatedThreadResponse{
		Data: threadResponses,
		Meta: commonDto.PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  total,
			Limit:       limit,
		},
	}, nil
}

func (s *service) GetThreadsByUsername(ctx context.Context, currentUserID uuid.UUID, username string, page, limit int) (*commonDto.PaginatedThreadResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	currentUser, err := s.userRepo.FindByID(ctx, currentUserID.String())
	if err != nil {
		return nil, fmt.Errorf("current user not found: %w", apperror.ErrNotFound)
	}

	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", apperror.ErrNotFound)
	}

	allowedAudiences := s.determineAllowedAudiences(currentUser.Role.Name)

	offset := (page - 1) * limit
	threads, total, err := s.threadRepo.FindByUserID(ctx, user.ID, allowedAudiences, offset, limit)
	if err != nil {
		return nil, err
	}

	var threadResponses []commonDto.ThreadResponse
	for _, thread := range threads {
		threadResponses = append(threadResponses, s.buildThreadResponse(ctx, *thread, &currentUserID))
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &commonDto.PaginatedThreadResponse{
		Data: threadResponses,
		Meta: commonDto.PaginationMeta{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  total,
			Limit:       limit,
		},
	}, nil
}

func (s *service) GetThreadBySlug(ctx context.Context, slug string) (*commonDto.ThreadResponse, error) {
	thread, err := s.threadRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("thread not found: %w", apperror.ErrNotFound)
	}

	resp := s.buildThreadResponse(ctx, *thread, nil)
	return &resp, nil
}

func (s *service) IncrementView(ctx context.Context, threadID uuid.UUID, userID uuid.UUID) error {
	return s.viewService.IncrementView(ctx, threadID, userID)
}

func (s *service) DeleteThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error {
	thread, err := s.threadRepo.FindByID(ctx, threadID)
	if err != nil {
		return fmt.Errorf("thread not found: %w", apperror.ErrNotFound)
	}

	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("user not found: %w", apperror.ErrNotFound)
	}

	if thread.UserID != userID && user.Role.Name != entity.RoleAdmin {
		return fmt.Errorf("unauthorized: you can only delete your own threads unless you are an admin: %w", apperror.ErrForbidden)
	}

	for _, att := range thread.Attachments {
		_ = s.fileStorage.DeleteImage(ctx, att.FileURL)
		if err := s.attachmentRepo.Delete(ctx, att.ID); err != nil {
			return fmt.Errorf("failed to delete attachment record: %w", err)
		}
	}

	if err := s.threadRepo.Delete(ctx, threadID); err != nil {
		return err
	}

	if s.meili != nil {
		_ = s.meili.DeleteThread(threadID.String())
	}

	return nil
}

func (s *service) UpdateThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID, req threadDto.UpdateThreadRequest) error {
	thread, err := s.threadRepo.FindByID(ctx, threadID)
	if err != nil {
		return fmt.Errorf("thread not found: %w", apperror.ErrNotFound)
	}

	if thread.UserID != userID {
		return fmt.Errorf("unauthorized: you can only update your own thread: %w", apperror.ErrForbidden)
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return fmt.Errorf("invalid category id format: %w", apperror.ErrBadRequest)
	}

	thread.Title = req.Title
	thread.Content = req.Content
	thread.CategoryID = &categoryID

	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("user not found: %w", apperror.ErrNotFound)
	}

	if err := s.validateAudienceForRole(user.Role.Name, req.Audience); err != nil {
		return err
	}
	thread.Audience = req.Audience

	// Update Attachments: Delete removed ones
	currentAttachments := make(map[uint]entity.Attachment)
	for _, att := range thread.Attachments {
		currentAttachments[att.ID] = att
	}
	desiredAttachments := make(map[uint]bool)
	for _, id := range req.AttachmentIDs {
		desiredAttachments[id] = true
	}

	for id, att := range currentAttachments {
		if !desiredAttachments[id] {
			_ = s.fileStorage.DeleteImage(ctx, att.FileURL)
			_ = s.attachmentRepo.Delete(ctx, id)
		}
	}

	if err := s.processThreadAttachments(ctx, thread.ID, userID, req.AttachmentIDs); err != nil {
		return err
	}

	if err := s.threadRepo.Update(ctx, thread); err != nil {
		return err
	}

	if s.meili != nil {
		reloadedThread, err := s.threadRepo.FindByID(ctx, threadID)
		if err == nil {
			_ = s.meili.IndexThread(reloadedThread)
		}
	}

	return nil
}
