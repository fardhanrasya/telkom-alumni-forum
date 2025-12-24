package service

import (
	"context"
	"mime/multipart"
	"time"

	"anoa.com/telkomalumiforum/internal/dto"
	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/google/uuid"
)

type AttachmentService interface {
	UploadAttachment(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader) (*dto.UploadAttachmentResponse, error)
	CleanupOrphanAttachments(ctx context.Context) error
}

type attachmentService struct {
	attachmentRepo repository.AttachmentRepository
	fileStorage    storage.ImageStorage
}

func NewAttachmentService(attachmentRepo repository.AttachmentRepository, fileStorage storage.ImageStorage) AttachmentService {
	return &attachmentService{
		attachmentRepo: attachmentRepo,
		fileStorage:    fileStorage,
	}
}

func (s *attachmentService) UploadAttachment(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader) (*dto.UploadAttachmentResponse, error) {
	f, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	url, err := s.fileStorage.UploadImage(ctx, f, "attachments", file.Filename)
	if err != nil {
		return nil, err
	}

	attachment := &model.Attachment{
		UserID:   userID,
		FileURL:  url,
		FileType: file.Header.Get("Content-Type"),
		// ThreadID and PostID are nil by default
	}

	if err := s.attachmentRepo.Create(ctx, attachment); err != nil {
		return nil, err
	}

	return &dto.UploadAttachmentResponse{
		ID:       attachment.ID,
		FileURL:  attachment.FileURL,
		FileType: attachment.FileType,
	}, nil
}

func (s *attachmentService) CleanupOrphanAttachments(ctx context.Context) error {
	// Cutoff time: 24 hours ago
	cutoff := time.Now().Add(-24 * time.Hour)

	orphans, err := s.attachmentRepo.FindOrphans(ctx, cutoff)
	if err != nil {
		return err
	}

	for _, orphan := range orphans {
		// 1. Delete from Cloudinary
		if err := s.fileStorage.DeleteImage(ctx, orphan.FileURL); err != nil {
			// e.g., print error but continue with other files?
			// In a real app we'd use a logger.
		}

		// 2. Delete from DB
		if err := s.attachmentRepo.Delete(ctx, orphan.ID); err != nil {
			// If DB delete fails, next run will pick it up again.
		}
	}
	return nil
}
