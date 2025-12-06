package service

import (
	"context"
	"fmt"
	"strings"

	"anoa.com/telkomalumiforum/internal/dto"
	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"anoa.com/telkomalumiforum/pkg/storage"
	"github.com/google/uuid"
)

type ThreadService interface {
	CreateThread(ctx context.Context, userID uuid.UUID, req dto.CreateThreadRequest) error
}

type threadService struct {
	threadRepo  repository.ThreadRepository
	categoryRepo repository.CategoryRepository
	fileStorage storage.ImageStorage
}

func NewThreadService(threadRepo repository.ThreadRepository, categoryRepo repository.CategoryRepository, fileStorage storage.ImageStorage) ThreadService {
	return &threadService{
		threadRepo:   threadRepo,
		categoryRepo: categoryRepo,
		fileStorage:  fileStorage,
	}
}

func (s *threadService) CreateThread(ctx context.Context, userID uuid.UUID, req dto.CreateThreadRequest) error {
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return fmt.Errorf("invalid category id format")
	}

	category, err := s.categoryRepo.FindByID(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("invalid category id")
	}

	slug := strings.ReplaceAll(strings.ToLower(req.Title), " ", "-")
	
	// Basic slug uniqueness check
	existing, _ := s.threadRepo.FindBySlug(ctx, slug)
	if existing != nil {
		// Append a short random string or timestamp for uniqueness
		slug = fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])
	}

	thread := &model.Thread{
		CategoryID: &category.ID,
		UserID:     userID,
		Title:      req.Title,
		Slug:       slug,
		Content:    req.Content,
		Audience:   req.Audience,
	}

	var attachments []model.Attachment
	for _, file := range req.Attachments {
		f, err := file.Open()
		if err != nil {
			return err
		}
		defer f.Close()

		url, err := s.fileStorage.UploadImage(ctx, f, "threads", file.Filename)
		if err != nil {
			return err
		}

		attachments = append(attachments, model.Attachment{
			UserID:   userID,
			FileURL:  url,
			FileType: file.Header.Get("Content-Type"),
		})
	}

	thread.Attachments = attachments

	return s.threadRepo.Create(ctx, thread)
}
