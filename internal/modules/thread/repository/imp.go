package thread

import (
	"context"

	"anoa.com/telkomalumiforum/internal/entity"
	"github.com/google/uuid"
)

func (r *repository) GetTrending(ctx context.Context, limit int) ([]*entity.Thread, error) {
	var ids []uuid.UUID

	query := `
		SELECT id
		FROM threads
		WHERE created_at >= NOW() - INTERVAL '7 days'
		ORDER BY (
			(COALESCE(views, 0) + 
			((SELECT COUNT(*) FROM thread_likes WHERE thread_likes.thread_id = threads.id) * 5) + 
			(COALESCE(replies_count, 0) * 30))
			/ 
			POWER((EXTRACT(EPOCH FROM (NOW() - created_at))/3600) + 2, 1.8)
		) DESC
		LIMIT ?
	`

	if err := r.db.WithContext(ctx).Raw(query, limit).Scan(&ids).Error; err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return []*entity.Thread{}, nil
	}

	var threads []*entity.Thread
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("User").
		Preload("User.Profile").
		Preload("Attachments").
		Where("id IN ?", ids).
		Find(&threads).Error; err != nil {
		return nil, err
	}

	// Reorder threads to match the trending order
	threadMap := make(map[uuid.UUID]*entity.Thread)
	for _, t := range threads {
		threadMap[t.ID] = t
	}

	orderedThreads := make([]*entity.Thread, 0, len(ids))
	for _, id := range ids {
		if t, ok := threadMap[id]; ok {
			orderedThreads = append(orderedThreads, t)
		}
	}

	return orderedThreads, nil
}
