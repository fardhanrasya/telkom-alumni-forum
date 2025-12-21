package repository

import (
	"context"

	"anoa.com/telkomalumiforum/internal/model"
	"github.com/google/uuid"
)

func (r *threadRepository) GetTrending(ctx context.Context, limit int) ([]*model.Thread, error) {
	var ids []uuid.UUID

	query := `
		SELECT id
		FROM threads
		WHERE created_at >= NOW() - INTERVAL '7 days'
		ORDER BY (
			(COALESCE(views, 0) + 
			((SELECT COUNT(*) FROM thread_likes WHERE thread_likes.thread_id = threads.id) * 10))
			/ 
			POWER((EXTRACT(EPOCH FROM (NOW() - created_at))/3600) + 2, 1.8)
		) DESC
		LIMIT ?
	`

	if err := r.db.WithContext(ctx).Raw(query, limit).Scan(&ids).Error; err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return []*model.Thread{}, nil
	}

	var threads []*model.Thread
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
	threadMap := make(map[uuid.UUID]*model.Thread)
	for _, t := range threads {
		threadMap[t.ID] = t
	}

	orderedThreads := make([]*model.Thread, 0, len(ids))
	for _, id := range ids {
		if t, ok := threadMap[id]; ok {
			orderedThreads = append(orderedThreads, t)
		}
	}

	return orderedThreads, nil
}
