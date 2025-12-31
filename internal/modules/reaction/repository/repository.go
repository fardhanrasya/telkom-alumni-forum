package repository

import (
	"context"

	"anoa.com/telkomalumiforum/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReactionRepository interface {
	// Toggle: returns oldEmoji (if any), newEmoji (if any), error
	ToggleReaction(ctx context.Context, reaction *entity.Reaction) (string, string, error)
	GetUserReactions(ctx context.Context, userID uuid.UUID, refID uuid.UUID, refType string) ([]string, error)
	GetReactionsCount(ctx context.Context, refID uuid.UUID, refType string) (map[string]int64, error)
}

type reactionRepository struct {
	db *gorm.DB
}

func NewReactionRepository(db *gorm.DB) ReactionRepository {
	return &reactionRepository{db: db}
}

func (r *reactionRepository) ToggleReaction(ctx context.Context, reaction *entity.Reaction) (string, string, error) {
	// Use Find with slice to avoid "record not found" log noise from GORM's First()
	var existing []entity.Reaction
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND reference_id = ? AND reference_type = ?",
			reaction.UserID, reaction.ReferenceID, reaction.ReferenceType).
		Limit(1).
		Find(&existing).Error

	if err != nil {
		return "", "", err
	}

	if len(existing) > 0 {
		// Reaction Exists
		record := existing[0]
		oldEmoji := record.Emoji

		if record.Emoji == reaction.Emoji {
			// Clicked same emoji -> Toggle Off (Delete)
			if err := r.db.WithContext(ctx).Delete(&record).Error; err != nil {
				return "", "", err
			}
			// Removed: oldEmoji returned, newEmoji empty
			return oldEmoji, "", nil
		} else {
			// Clicked different emoji -> Replace (Update)
			record.Emoji = reaction.Emoji
			if err := r.db.WithContext(ctx).Save(&record).Error; err != nil {
				return "", "", err
			}
			// Replaced: oldEmoji returned, newEmoji returned
			return oldEmoji, reaction.Emoji, nil
		}
	} else {
		// No Reaction Found -> Create (Like)
		if err := r.db.WithContext(ctx).Create(reaction).Error; err != nil {
			return "", "", err
		}
		// Added: oldEmoji empty, newEmoji returned
		return "", reaction.Emoji, nil
	}
}

func (r *reactionRepository) GetUserReactions(ctx context.Context, userID uuid.UUID, refID uuid.UUID, refType string) ([]string, error) {
	var emojis []string
	err := r.db.WithContext(ctx).
		Model(&entity.Reaction{}).
		Where("user_id = ? AND reference_id = ? AND reference_type = ?", userID, refID, refType).
		Pluck("emoji", &emojis).Error
	return emojis, err
}

func (r *reactionRepository) GetReactionsCount(ctx context.Context, refID uuid.UUID, refType string) (map[string]int64, error) {
	type Result struct {
		Emoji string
		Count int64
	}
	var results []Result

	err := r.db.WithContext(ctx).
		Model(&entity.Reaction{}).
		Select("emoji, count(*) as count").
		Where("reference_id = ? AND reference_type = ?", refID, refType).
		Group("emoji").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, res := range results {
		counts[res.Emoji] = res.Count
	}
	return counts, nil
}
