package service

import (
	"context"

	"anoa.com/telkomalumiforum/internal/dto"
)

func (s *threadService) GetTrendingThreads(ctx context.Context, limit int) ([]dto.ThreadResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	threads, err := s.threadRepo.GetTrending(ctx, limit)
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

		// Note: We might want to optimize this by fetching likes count in bulk or joining in the repo query.
		// However, for trending threads (usually small limit like 10), calling GetThreadLikes N times is acceptable for now.
		// A better approach would be to include likes count in the GetTrending repo query result,
		// but that requires changing the Thread model or returning a different struct.
		// Since we want standard ThreadResponse, let's keep it simple.
		likesCount, _ := s.likeService.GetThreadLikes(ctx, thread.ID)

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
			CreatedAt:    thread.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		threadResponses = append(threadResponses, resp)
	}

	return threadResponses, nil
}
