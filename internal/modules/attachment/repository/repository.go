package repository

import (
	"context"

	"time"

	"anoa.com/telkomalumiforum/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AttachmentRepository interface {
	Create(ctx context.Context, attachment *entity.Attachment) error
	UpdateThreadID(ctx context.Context, attachmentIDs []uint, threadID uuid.UUID, userID uuid.UUID) error
	UpdatePostID(ctx context.Context, attachmentIDs []uint, postID uuid.UUID, userID uuid.UUID) error
	FindOrphans(ctx context.Context, cutoffTime time.Time) ([]entity.Attachment, error)
	Delete(ctx context.Context, id uint) error
}

type attachmentRepository struct {
	db *gorm.DB
}

func NewAttachmentRepository(db *gorm.DB) AttachmentRepository {
	return &attachmentRepository{db: db}
}

func (r *attachmentRepository) Create(ctx context.Context, attachment *entity.Attachment) error {
	return r.db.WithContext(ctx).Create(attachment).Error
}

func (r *attachmentRepository) UpdateThreadID(ctx context.Context, attachmentIDs []uint, threadID uuid.UUID, userID uuid.UUID) error {
	// Only allow updating if:
	// 1. Owned by user (user_id = ?)
	// 2. Not attached to another thread (thread_id IS NULL OR thread_id = ?)
	// 3. Not attached to a post (post_id IS NULL)
	return r.db.WithContext(ctx).Model(&entity.Attachment{}).
		Where("id IN ? AND user_id = ?", attachmentIDs, userID).
		Where("(thread_id IS NULL OR thread_id = ?) AND post_id IS NULL", threadID).
		Update("thread_id", threadID).Error
}

func (r *attachmentRepository) UpdatePostID(ctx context.Context, attachmentIDs []uint, postID uuid.UUID, userID uuid.UUID) error {
	// Only allow updating if:
	// 1. Owned by user
	// 2. Not attached to a thread (thread_id IS NULL)
	// 3. Not attached to another post (post_id IS NULL OR post_id = ?)
	return r.db.WithContext(ctx).Model(&entity.Attachment{}).
		Where("id IN ? AND user_id = ?", attachmentIDs, userID).
		Where("thread_id IS NULL AND (post_id IS NULL OR post_id = ?)", postID).
		Update("post_id", postID).Error
}

func (r *attachmentRepository) FindOrphans(ctx context.Context, cutoffTime time.Time) ([]entity.Attachment, error) {
	var attachments []entity.Attachment
	err := r.db.WithContext(ctx).
		Where("thread_id IS NULL AND post_id IS NULL AND created_at < ?", cutoffTime).
		Find(&attachments).Error
	return attachments, err
}

func (r *attachmentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&entity.Attachment{}, id).Error
}
