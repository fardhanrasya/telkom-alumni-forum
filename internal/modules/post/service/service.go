package post

import (
	"context"
	"fmt"
	"time"

	"anoa.com/telkomalumiforum/internal/entity"
	attachmentRepo "anoa.com/telkomalumiforum/internal/modules/attachment/repository"
	leaderboard "anoa.com/telkomalumiforum/internal/modules/leaderboard/service"
	notifService "anoa.com/telkomalumiforum/internal/modules/notification/service"
	postDto "anoa.com/telkomalumiforum/internal/modules/post/dto"
	postRepo "anoa.com/telkomalumiforum/internal/modules/post/repository"
	reaction "anoa.com/telkomalumiforum/internal/modules/reaction/service"
	search "anoa.com/telkomalumiforum/internal/modules/search/service"
	repo "anoa.com/telkomalumiforum/internal/modules/thread/repository"
	userRepo "anoa.com/telkomalumiforum/internal/modules/user/repository"
	"anoa.com/telkomalumiforum/pkg/dto"
	"anoa.com/telkomalumiforum/pkg/ratelimiter"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type PostService interface {
	CreatePost(ctx context.Context, userID uuid.UUID, req postDto.CreatePostRequest) (*postDto.PostResponse, error)
	GetPostsByThreadID(ctx context.Context, threadID uuid.UUID, userID *uuid.UUID, filter postDto.PostFilter) (*postDto.PaginatedPostResponse, error)
	GetPostByID(ctx context.Context, postID uuid.UUID, userID *uuid.UUID) (*postDto.PostResponse, error)
	UpdatePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID, req postDto.UpdatePostRequest) (*postDto.PostResponse, error)
	DeletePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
}

type postService struct {
	postRepo            postRepo.PostRepository
	threadRepo          repo.Repository
	userRepo            userRepo.UserRepository
	attachmentRepo      attachmentRepo.AttachmentRepository
	reactionService     reaction.ReactionService
	fileStorage         storage.ImageStorage
	redisClient         *redis.Client
	notificationService notifService.NotificationService
	meili               search.MeiliSearchService
	leaderboardService  leaderboard.LeaderboardService
}

func NewPostService(postRepo postRepo.PostRepository, threadRepo repo.Repository, userRepo userRepo.UserRepository, attachmentRepo attachmentRepo.AttachmentRepository, reactionService reaction.ReactionService, fileStorage storage.ImageStorage, redisClient *redis.Client, notificationService notifService.NotificationService, meili search.MeiliSearchService, leaderboardService leaderboard.LeaderboardService) PostService {
	return &postService{
		postRepo:            postRepo,
		threadRepo:          threadRepo,
		userRepo:            userRepo,
		attachmentRepo:      attachmentRepo,
		reactionService:     reactionService,
		fileStorage:         fileStorage,
		redisClient:         redisClient,
		notificationService: notificationService,
		meili:               meili,
		leaderboardService:  leaderboardService,
	}
}

func (s *postService) CreatePost(ctx context.Context, userID uuid.UUID, req postDto.CreatePostRequest) (*postDto.PostResponse, error) {
	// Global Cooldown
	globalLimit := ratelimiter.GetDurationFromEnv("RATE_LIMIT_GLOBAL", 5*time.Second)
	allowed, err := ratelimiter.CheckAndSetRateLimit(ctx, s.redisClient, userID, "global", globalLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !allowed {
		ttl, _ := ratelimiter.GetRateLimitTTL(ctx, s.redisClient, userID, "global")
		return nil, &ratelimiter.RateLimitError{
			Message:    fmt.Sprintf("you are doing that too fast. Please wait %.0f seconds", ttl.Seconds()),
			RetryAfter: ttl,
		}
	}

	// 2. Post-specific Cooldown
	postLimit := ratelimiter.GetDurationFromEnv("RATE_LIMIT_POST", 15*time.Second)
	allowed, err = ratelimiter.CheckAndSetRateLimit(ctx, s.redisClient, userID, "post", postLimit)
	if err != nil {
		_ = ratelimiter.ClearRateLimit(ctx, s.redisClient, userID, "global") // Rollback global
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !allowed {
		_ = ratelimiter.ClearRateLimit(ctx, s.redisClient, userID, "global") // Rollback global
		ttl, _ := ratelimiter.GetRateLimitTTL(ctx, s.redisClient, userID, "post")
		return nil, &ratelimiter.RateLimitError{
			Message:    fmt.Sprintf("you can only create one post every %.0f seconds. Please wait %.0f seconds", postLimit.Seconds(), ttl.Seconds()),
			RetryAfter: ttl,
		}
	}

	// Defer rollback in case of creation failure
	creationFailed := true
	defer func() {
		if creationFailed {
			_ = ratelimiter.ClearRateLimit(ctx, s.redisClient, userID, "global")
			_ = ratelimiter.ClearRateLimit(ctx, s.redisClient, userID, "post")
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

	post := &entity.Post{
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

	go func() {
		// Avoid notifying the user themselves
		var targetUserID uuid.UUID
		var notifType string
		var message string

		if parentID != nil {
			// Reply to a post
			// Re-fetch parent to safely check UserID
			p, err := s.postRepo.FindByID(context.Background(), *parentID)
			if err == nil && p.UserID != userID {
				targetUserID = p.UserID
				notifType = "reply_post"
				message = fmt.Sprintf("Someone replied to your post in '%s'", thread.Title)
			}
		} else {
			// Reply to thread
			if thread.UserID != userID {
				targetUserID = thread.UserID
				notifType = "reply_thread"
				message = fmt.Sprintf("Someone commented on your thread '%s'", thread.Title)
			}
		}

		if targetUserID != uuid.Nil {
			notification := &entity.Notification{
				UserID:     targetUserID,
				ActorID:    userID,
				EntityID:   post.ID, // The new reply
				EntitySlug: thread.Slug,
				EntityType: "post",
				Type:       notifType,
				Message:    message,
			}
			_ = s.notificationService.CreateNotification(context.Background(), notification)
		}

		// Gamification: Give points to thread author (Engagement) - nil actorID for comment events
		if thread.UserID != userID && s.leaderboardService != nil {
			s.leaderboardService.AddGamificationPointsAsync(thread.UserID, leaderboard.ActionCommentReceived, post.ID.String(), "posts", nil)
		}
	}()

	// Index to Meilisearch
	post.Thread = *thread
	// Ensure User is populated if not already
	if post.User.Username == "" {
		u, _ := s.userRepo.FindByID(ctx, userID.String())
		if u != nil {
			post.User = *u
		}
	}

	if s.meili != nil {
		if err := s.meili.IndexPost(post); err != nil {
			fmt.Printf("Failed to index post: %v\n", err)
		}
	}

	return s.mapToResponse(ctx, post, &userID), nil
}

func (s *postService) GetPostsByThreadID(ctx context.Context, threadID uuid.UUID, userID *uuid.UUID, filter postDto.PostFilter) (*postDto.PaginatedPostResponse, error) {
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}

	// Fetch ALL posts for the thread to build the tree
	allPosts, err := s.postRepo.FindAllByThreadID(ctx, threadID)
	if err != nil {
		return nil, err
	}

	// 1. Convert all to DTOs and store in map
	postMap := make(map[uuid.UUID]*postDto.PostResponse)
	for _, p := range allPosts {
		postMap[p.ID] = s.mapToResponse(ctx, p, userID)
	}

	// 2. Build Tree
	var roots []*postDto.PostResponse
	for _, p := range allPosts {
		node := postMap[p.ID]
		if p.ParentID == nil {
			roots = append(roots, node)
		} else {
			if parent, exists := postMap[*p.ParentID]; exists {
				parent.Replies = append(parent.Replies, node)
			}
			// If parent doesn't exist (shouldn't happen with valid FKs), we ignore or treat as root.
			// Currently ignoring to avoid clutter.
		}
	}

	// 3. Paginate Roots
	totalRoots := int64(len(roots))
	startIndex := (filter.Page - 1) * filter.Limit
	endIndex := startIndex + filter.Limit

	if startIndex < 0 {
		startIndex = 0
	}

	var paginatedRoots []postDto.PostResponse
	if startIndex < int(totalRoots) {
		if endIndex > int(totalRoots) {
			endIndex = int(totalRoots)
		}
		for _, r := range roots[startIndex:endIndex] {
			paginatedRoots = append(paginatedRoots, *r)
		}
	} else {
		paginatedRoots = []postDto.PostResponse{}
	}

	totalPages := int(totalRoots) / filter.Limit
	if int(totalRoots)%filter.Limit != 0 {
		totalPages++
	}

	return &postDto.PaginatedPostResponse{
		Data: paginatedRoots,
		Meta: dto.PaginationMeta{
			CurrentPage: filter.Page,
			TotalPages:  totalPages,
			TotalItems:  totalRoots,
			Limit:       filter.Limit,
		},
	}, nil
}

func (s *postService) GetPostByID(ctx context.Context, postID uuid.UUID, userID *uuid.UUID) (*postDto.PostResponse, error) {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	return s.mapToResponse(ctx, post, userID), nil
}

func (s *postService) UpdatePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID, req postDto.UpdatePostRequest) (*postDto.PostResponse, error) {
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
	currentAttachments := make(map[uint]entity.Attachment)
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

	// Index to Meilisearch
	if s.meili != nil {
		// Ensure Thread is loaded for IndexPost (Audience check)
		if post.Thread.ID == uuid.Nil {
			t, err := s.threadRepo.FindByID(ctx, post.ThreadID)
			if err == nil {
				post.Thread = *t
			}
		}
		_ = s.meili.IndexPost(post)
	}

	return s.mapToResponse(ctx, post, &userID), nil
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

	if err := s.postRepo.Delete(ctx, postID); err != nil {
		return err
	}

	if s.meili != nil {
		_ = s.meili.DeletePost(postID.String())
	}

	return nil
}

