package thread

import (
	"context"

	"anoa.com/telkomalumiforum/pkg/dto"
)

func (s *service) GetTrendingThreads(ctx context.Context, limit int) ([]dto.ThreadResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	threads, err := s.threadRepo.GetTrending(ctx, limit)
	if err != nil {
		return nil, err
	}

	var threadResponses []dto.ThreadResponse
	for _, thread := range threads {
		// Pass nil for currentUserID as this is a public trending list
		threadResponses = append(threadResponses, s.buildThreadResponse(ctx, *thread, nil))
	}

	return threadResponses, nil
}
