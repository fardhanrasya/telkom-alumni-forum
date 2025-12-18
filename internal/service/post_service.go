package service

import (
	"context"
	"fmt"
	"time"

	"anoa.com/telkomalumiforum/internal/dto"
	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type PostService interface {
	CreatePost(ctx context.Context, userID uuid.UUID, req dto.CreatePostRequest) (*dto.PostResponse, error)
	GetPostsByThreadID(ctx context.Context, threadID uuid.UUID, filter dto.PostFilter) (*dto.PaginatedPostResponse, error)
	GetPostByID(ctx context.Context, postID uuid.UUID) (*dto.PostResponse, error)
	UpdatePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID, req dto.UpdatePostRequest) (*dto.PostResponse, error)
	DeletePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
}

type postService struct {
	postRepo       repository.PostRepository
	threadRepo     repository.ThreadRepository
	userRepo       repository.UserRepository
	attachmentRepo repository.AttachmentRepository
	likeService    LikeService
	fileStorage    storage.ImageStorage
	redisClient    *redis.Client
}

func NewPostService(postRepo repository.PostRepository, threadRepo repository.ThreadRepository, userRepo repository.UserRepository, attachmentRepo repository.AttachmentRepository, likeService LikeService, fileStorage storage.ImageStorage, redisClient *redis.Client) PostService {
	return &postService{
		postRepo:       postRepo,
		threadRepo:     threadRepo,
		userRepo:       userRepo,
		attachmentRepo: attachmentRepo,
		likeService:    likeService,
		fileStorage:    fileStorage,
		redisClient:    redisClient,
	}
}

func (s *postService) CreatePost(ctx context.Context, userID uuid.UUID, req dto.CreatePostRequest) (*dto.PostResponse, error) {
	// Global Cooldown: 5 seconds
	allowed, err := CheckAndSetRateLimit(ctx, s.redisClient, userID, "global", 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !allowed {
		ttl, _ := GetRateLimitTTL(ctx, s.redisClient, userID, "global")
		return nil, &RateLimitError{
			Message:    fmt.Sprintf("you are doing that too fast. Please wait %.0f seconds", ttl.Seconds()),
			RetryAfter: ttl,
		}
	}

	// 2. Post-specific Cooldown: 15 seconds
	allowed, err = CheckAndSetRateLimit(ctx, s.redisClient, userID, "post", 15*time.Second)
	if err != nil {
		_ = ClearRateLimit(ctx, s.redisClient, userID, "global") // Rollback global
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !allowed {
		_ = ClearRateLimit(ctx, s.redisClient, userID, "global") // Rollback global
		ttl, _ := GetRateLimitTTL(ctx, s.redisClient, userID, "post")
		return nil, &RateLimitError{
			Message:    fmt.Sprintf("you can only create one post every 15 seconds. Please wait %.0f seconds", ttl.Seconds()),
			RetryAfter: ttl,
		}
	}

	// Defer rollback in case of creation failure
	creationFailed := true
	defer func() {
		if creationFailed {
			_ = ClearRateLimit(ctx, s.redisClient, userID, "global")
			_ = ClearRateLimit(ctx, s.redisClient, userID, "post")
		}
	}()

	threadID, err := uuid.Parse(req.ThreadID)
	if err != nil {
		return nil, fmt.Errorf("invalid thread id")
	}

	// Verify Thread Exists
	thread, err := s.threadRepo.FindByID(ctx, threadID)
	if err != nil || thread == nil {
		return nil, fmt.Errorf("thread not found")
	}

	var parentID *uuid.UUID
	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent id")
		}
		// Verify Parent Exists
		parent, err := s.postRepo.FindByID(ctx, pid)
		if err != nil || parent == nil {
			return nil, fmt.Errorf("parent post not found")
		}
		parentID = &pid
	}

	post := &model.Post{
		ThreadID: threadID,
		UserID:   userID,
		ParentID: parentID,
		Content:  req.Content,
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, err
	}

	if len(req.AttachmentIDs) > 0 {
		if err := s.attachmentRepo.UpdatePostID(ctx, req.AttachmentIDs, post.ID, userID); err != nil {
			return nil, err
		}

		// Reload post to get attachments
		reloaded, err := s.postRepo.FindByID(ctx, post.ID)
		if err == nil {
			post = reloaded
		}
	} else {
		// Just load user for response construction
		user, _ := s.userRepo.FindByID(ctx, userID.String())
		post.User = *user
	}

	// Everything succeeded, don't roll back the rate limits.
	creationFailed = false

	return s.mapToResponse(post), nil
}

func (s *postService) GetPostsByThreadID(ctx context.Context, threadID uuid.UUID, filter dto.PostFilter) (*dto.PaginatedPostResponse, error) {
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}

	offset := (filter.Page - 1) * filter.Limit

	posts, total, err := s.postRepo.FindByThreadID(ctx, threadID, offset, filter.Limit)
	if err != nil {
		return nil, err
	}

	var responses []dto.PostResponse
	for _, p := range posts {
		responses = append(responses, *s.mapToResponse(p))
	}

	// Create empty slice if nil to ensure JSON array output [] instead of null
	if responses == nil {
		responses = []dto.PostResponse{}
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit != 0 {
		totalPages++
	}

	return &dto.PaginatedPostResponse{
		Data: responses,
		Meta: dto.PaginationMeta{
			CurrentPage: filter.Page,
			TotalPages:  totalPages,
			TotalItems:  total,
			Limit:       filter.Limit,
		},
	}, nil
}

func (s *postService) GetPostByID(ctx context.Context, postID uuid.UUID) (*dto.PostResponse, error) {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	return s.mapToResponse(post), nil
}

func (s *postService) UpdatePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID, req dto.UpdatePostRequest) (*dto.PostResponse, error) {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	if post.UserID != userID {
		return nil, fmt.Errorf("unauthorized: you can only update your own post")
	}

	post.Content = req.Content
	// Update Attachments
	// 1. Identify which attachments to keep vs delete
	currentAttachments := make(map[uint]model.Attachment)
	for _, att := range post.Attachments {
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

	// Add new attachments
	if len(req.AttachmentIDs) > 0 {
		if err := s.attachmentRepo.UpdatePostID(ctx, req.AttachmentIDs, post.ID, userID); err != nil {
			return nil, err
		}
	}

	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, err
	}

	// Reload to get updated attachments for response
	updatedPost, err := s.postRepo.FindByID(ctx, post.ID)
	if err == nil {
		post = updatedPost
	}

	return s.mapToResponse(post), nil
}

func (s *postService) DeletePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return err
	}

	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if post.UserID != userID && user.Role.Name != "admin" {
		return fmt.Errorf("unauthorized: you can only delete your own post unless you are an admin")
	}

	// Delete Attachments
	for _, att := range post.Attachments {
		_ = s.fileStorage.DeleteImage(ctx, att.FileURL)
		_ = s.attachmentRepo.Delete(ctx, att.ID)
	}

	// Since we set CASCADE on ParentID in database (scheme.sql), child posts *should* be deleted automatically?
	// Scheme: parent_id UUID REFERENCES posts(id) ON DELETE CASCADE
	// Yes, deletions cascade.

	return s.postRepo.Delete(ctx, postID)
}

func (s *postService) mapToResponse(post *model.Post) *dto.PostResponse {
	var attachments []dto.AttachmentResponse
	for _, att := range post.Attachments {
		attachments = append(attachments, dto.AttachmentResponse{
			ID:       att.ID,
			FileURL:  att.FileURL,
			FileType: att.FileType,
		})
	}

	authorName := "Unknown"
	if post.User.Username != "" {
		authorName = post.User.Username
	}

	likesCount, _ := s.likeService.GetPostLikes(context.Background(), post.ID)

	return &dto.PostResponse{
		ID:          post.ID,
		ThreadID:    post.ThreadID,
		ParentID:    post.ParentID,
		Content:     post.Content,
		Author:      authorName,
		Attachments: attachments,
		LikesCount:  likesCount,
		CreatedAt:   post.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   post.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
