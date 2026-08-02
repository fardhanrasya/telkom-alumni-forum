package thread

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"anoa.com/telkomalumiforum/internal/entity"
	cosmeticDto "anoa.com/telkomalumiforum/internal/modules/cosmetic/dto"
	"anoa.com/telkomalumiforum/pkg/apperror"
	commonDto "anoa.com/telkomalumiforum/pkg/dto"
	"anoa.com/telkomalumiforum/pkg/ratelimiter"
	"github.com/google/uuid"
)

// Helper methods to reduce code duplication in Service

func (s *service) buildThreadResponse(ctx context.Context, thread entity.Thread, currentUserID *uuid.UUID) commonDto.ThreadResponse {
	var attachments []commonDto.AttachmentResponse
	for _, att := range thread.Attachments {
		attachments = append(attachments, commonDto.AttachmentResponse{
			ID:       att.ID,
			FileURL:  att.FileURL,
			FileType: att.FileType,
		})
	}

	authorResponse := commonDto.AuthorResponse{
		Username: "Unknown",
	}
	if thread.User.Username != "" {
		authorResponse.Username = thread.User.Username
		authorResponse.AvatarURL = thread.User.AvatarURL
		authorResponse.Equip = cosmeticDto.FromEquipEntity(thread.User.Equip)
	}

	// Fetch reactions
	// Note: Ideally this should be batched or optimized in the future
	reactions, _ := s.reactionService.GetReactions(ctx, currentUserID, thread.ID, "thread")
	// If reactions is nil (error case handled gracefully in original code), provide empty default
	if reactions == nil {
		reactions = &commonDto.ReactionsResponse{
			Counts:      make(map[string]int64),
			UserReacted: nil,
		}
	}

	// Extract featured image from attachments or HTML content
	var firstImageURL *string
	for _, att := range thread.Attachments {
		if strings.HasPrefix(att.FileType, "image") || strings.Contains(att.FileURL, "cloudinary") || strings.Contains(att.FileURL, ".png") || strings.Contains(att.FileURL, ".jpg") || strings.Contains(att.FileURL, ".jpeg") || strings.Contains(att.FileURL, ".webp") {
			url := att.FileURL
			firstImageURL = &url
			break
		}
	}
	if firstImageURL == nil {
		re := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)
		matches := re.FindStringSubmatch(thread.Content)
		if len(matches) > 1 {
			imgSrc := matches[1]
			firstImageURL = &imgSrc
		}
	}

	return commonDto.ThreadResponse{
		ID:           thread.ID,
		CategoryName: thread.Category.Name,
		Title:        thread.Title,
		Slug:         thread.Slug,
		Content:      thread.Content,
		Audience:     thread.Audience,
		Views:        thread.Views,
		ReplyCount:   int64(thread.RepliesCount),
		Author:       authorResponse,
		ImageURL:     firstImageURL,
		Attachments:  attachments,
		Reactions:    *reactions,
		CreatedAt:    thread.CreatedAt.Format(time.RFC3339),
	}
}

func (s *service) checkCreateThreadRateLimit(ctx context.Context, userID uuid.UUID) (func(), error) {
	// 1. Global Cooldown
	// Use hardcoded fallback if env not set for now, typically these come from config injection but keeping simple as per request
	globalLimit := ratelimiter.GetDurationFromEnv("RATE_LIMIT_GLOBAL", 5*time.Second)
	allowed, err := ratelimiter.CheckAndSetRateLimit(ctx, s.redisClient, userID, ratelimiter.ScopeGlobal, globalLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !allowed {
		ttl, _ := ratelimiter.GetRateLimitTTL(ctx, s.redisClient, userID, ratelimiter.ScopeGlobal)
		return nil, &ratelimiter.RateLimitError{
			Message:    fmt.Sprintf("you are doing that too fast. Please wait %.0f seconds", ttl.Seconds()),
			RetryAfter: ttl,
		}
	}

	// 2. Thread-specific Cooldown
	threadLimit := ratelimiter.GetDurationFromEnv("RATE_LIMIT_THREAD", 5*time.Minute)
	allowed, err = ratelimiter.CheckAndSetRateLimit(ctx, s.redisClient, userID, ratelimiter.ScopeThread, threadLimit)
	if err != nil {
		_ = ratelimiter.ClearRateLimit(ctx, s.redisClient, userID, ratelimiter.ScopeGlobal) // Rollback global
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !allowed {
		_ = ratelimiter.ClearRateLimit(ctx, s.redisClient, userID, ratelimiter.ScopeGlobal) // Rollback global
		ttl, _ := ratelimiter.GetRateLimitTTL(ctx, s.redisClient, userID, ratelimiter.ScopeThread)
		return nil, &ratelimiter.RateLimitError{
			Message:    fmt.Sprintf("you can only create one thread every %.0f minutes. Please wait %.0f minutes", threadLimit.Minutes(), ttl.Minutes()),
			RetryAfter: ttl,
		}
	}

	// Return cleanup function to rollback if subsequent steps fail
	cleanup := func() {
		_ = ratelimiter.ClearRateLimit(ctx, s.redisClient, userID, ratelimiter.ScopeGlobal)
		_ = ratelimiter.ClearRateLimit(ctx, s.redisClient, userID, ratelimiter.ScopeThread)
	}

	return cleanup, nil
}

func (s *service) validateAudienceForRole(roleName, audience string) error {
	switch roleName {
	case entity.RoleSiswa:
		if audience == entity.AudienceGuru {
			return fmt.Errorf("%w: siswa cannot create/update thread for guru", apperror.ErrForbidden)
		}
	case entity.RoleGuru:
		if audience == entity.AudienceSiswa {
			return fmt.Errorf("%w: guru cannot create/update thread for siswa", apperror.ErrForbidden)
		}
	}
	return nil
}

func (s *service) generateUniqueSlug(ctx context.Context, title string) string {
	slug := strings.ToLower(title)
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
	return slug
}

func (s *service) determineAllowedAudiences(roleName string) []string {
	switch roleName {
	case entity.RoleSiswa:
		return []string{entity.AudienceSiswa, entity.AudienceSemua}
	case entity.RoleGuru:
		return []string{entity.AudienceGuru, entity.AudienceSemua}
	case "guest":
		return []string{entity.AudienceSemua}
	default:
		// Admin or others effectively see all
		return nil
	}
}

func (s *service) processThreadAttachments(ctx context.Context, threadID uuid.UUID, userID uuid.UUID, attachmentIDs []uint) error {
	if len(attachmentIDs) > 0 {
		return s.attachmentRepo.UpdateThreadID(ctx, attachmentIDs, threadID, userID)
	}
	return nil
}

func stripHTMLTags(html string) string {
	re := regexp.MustCompile("<[^>]*>")
	return re.ReplaceAllString(html, "")
}
