
package post

import (
	"context"

	
	postDto "anoa.com/telkomalumiforum/internal/modules/post/dto"
	"anoa.com/telkomalumiforum/internal/entity"
	"anoa.com/telkomalumiforum/pkg/dto"
	"github.com/google/uuid"
)

func (s *postService) mapToResponse(ctx context.Context, post *entity.Post) *postDto.PostResponse {
	var attachments []dto.AttachmentResponse
	for _, att := range post.Attachments {
		attachments = append(attachments, dto.AttachmentResponse{
			ID:       att.ID,
			FileURL:  att.FileURL,
			FileType: att.FileType,
		})
	}

	authorResponse := dto.AuthorResponse{
		Username: "Unknown",
	}
	if post.User.Username != "" {
		authorResponse.Username = post.User.Username
		authorResponse.AvatarURL = post.User.AvatarURL
	}

	// Fetch Reaction Data
	// Try to get user_id from context for "user_reacted" status
	var userIDPtr *uuid.UUID
	// Assuming the handler sets a "user_id" string in the context if utilizing Gin.
	if val := ctx.Value("user_id"); val != nil {
		if idStr, ok := val.(string); ok {
			if uid, err := uuid.Parse(idStr); err == nil {
				userIDPtr = &uid
			}
		} else if uid, ok := val.(uuid.UUID); ok {
			userIDPtr = &uid
		}
	}

	reactions, _ := s.reactionService.GetReactions(ctx, userIDPtr, post.ID, "post")
	likesCount := int64(reactions.Counts["👍"])

	return &postDto.PostResponse{
		ID:          post.ID,
		ThreadID:    post.ThreadID,
		ParentID:    post.ParentID,
		Content:     post.Content,
		Author:      authorResponse,
		Attachments: attachments,
		LikesCount:  likesCount,
		Reactions:   *reactions,
		CreatedAt:   post.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   post.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
