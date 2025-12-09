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
	GetAllThreads(ctx context.Context, userID uuid.UUID, filter dto.ThreadFilter) (*dto.PaginatedThreadResponse, error)
	DeleteThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error
}

type threadService struct {
	threadRepo     repository.ThreadRepository
	categoryRepo   repository.CategoryRepository
	userRepo       repository.UserRepository
	attachmentRepo repository.AttachmentRepository
	fileStorage    storage.ImageStorage
}

func NewThreadService(threadRepo repository.ThreadRepository, categoryRepo repository.CategoryRepository, userRepo repository.UserRepository, attachmentRepo repository.AttachmentRepository, fileStorage storage.ImageStorage) ThreadService {
	return &threadService{
		threadRepo:     threadRepo,
		categoryRepo:   categoryRepo,
		userRepo:       userRepo,
		attachmentRepo: attachmentRepo,
		fileStorage:    fileStorage,
	}
}

func (s *threadService) CreateThread(ctx context.Context, userID uuid.UUID, req dto.CreateThreadRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Validate Audience based on Role
	switch user.Role.Name {
	case "siswa":
		if req.Audience == "guru" {
			return fmt.Errorf("siswa cannot create thread for guru")
		}
	case "guru":
		if req.Audience == "siswa" {
			return fmt.Errorf("guru cannot create thread for siswa")
		}
	}

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

	if err := s.threadRepo.Create(ctx, thread); err != nil {
		return err
	}

	if len(req.AttachmentIDs) > 0 {
		if err := s.attachmentRepo.UpdateThreadID(ctx, req.AttachmentIDs, thread.ID, userID); err != nil {
			return err
		}
	}

	return nil
}

func (s *threadService) GetAllThreads(ctx context.Context, userID uuid.UUID, filter dto.ThreadFilter) (*dto.PaginatedThreadResponse, error) {
	// 1. Fetch User Role
	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// 2. Determine Allowed Audiences
	var effectiveAudiences []string
	var allowed []string

	switch user.Role.Name {
	case "siswa":
		allowed = []string{"siswa", "semua"}
	case "guru":
		allowed = []string{"guru", "semua"}
	default:
		// Admin sees everything, leave allowed empty to indicate "all" or specific logic
		// But Wait, if filter is set, we use filter. If not set, we return all. 
		// Existing logic: if len(audiences) > 0 -> WHERE audience IN...
		// So for admin, if we keep effectiveAudiences empty, it means no WHERE audience constraint -> ALL. 
	}
	
	if len(allowed) > 0 {
		if filter.Audience != "" {
			// Check if requested audience is allowed
			isAllowed := false
			for _, a := range allowed {
				if a == filter.Audience {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				return &dto.PaginatedThreadResponse{
					Data: []dto.ThreadResponse{},
					Meta: dto.PaginationMeta{
						CurrentPage: filter.Page,
						TotalPages:  0,
						TotalItems:  0,
						Limit:       filter.Limit,
					},
				}, nil
			}
			effectiveAudiences = []string{filter.Audience}
		} else {
			effectiveAudiences = allowed
		}
	} else {
		// Admin logic
		if filter.Audience != "" {
			effectiveAudiences = []string{filter.Audience}
		}
	}

	var categoryID *uuid.UUID
	if filter.CategoryID != "" {
		id, err := uuid.Parse(filter.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("invalid category id")
		}
		categoryID = &id
	}

	offset := (filter.Page - 1) * filter.Limit
	threads, total, err := s.threadRepo.FindAll(ctx, categoryID, filter.Search, effectiveAudiences, filter.SortBy, offset, filter.Limit)
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
		
		authorName := "Unknown"
		if thread.User.Username != "" {
			authorName = thread.User.Username
		}

		resp := dto.ThreadResponse{
			ID:           thread.ID,
			CategoryName: thread.Category.Name,
			Title:        thread.Title,
			Slug:         thread.Slug,
			Content:      thread.Content,
			Audience:     thread.Audience,
			Views:        thread.Views,
			Author:       authorName, 
			Attachments:  attachments,
			CreatedAt:    thread.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		threadResponses = append(threadResponses, resp)
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit != 0 {
		totalPages++
	}

	return &dto.PaginatedThreadResponse{
		Data: threadResponses,
		Meta: dto.PaginationMeta{
			CurrentPage: filter.Page,
			TotalPages:  totalPages,
			TotalItems:  total,
			Limit:       filter.Limit,
		},
	}, nil
}

func (s *threadService) DeleteThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error {
	// 1. Get Thread
	thread, err := s.threadRepo.FindByID(ctx, threadID)
	if err != nil {
		return err
	}

	// 2. Get Requesting User to check Role
	user, err := s.userRepo.FindByID(ctx, userID.String())
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// 3. Permission Check
	// Assuming "admin" is the role name for admin users.
	if thread.UserID != userID && user.Role.Name != "admin" {
		return fmt.Errorf("unauthorized: you can only delete your own threads unless you are an admin")
	}

	// 4. Delete Attachments
	for _, att := range thread.Attachments {
		// Delete from Cloudinary
		// We ignore error here to proceed with deletion, but ideally log it.
		// Since we don't have a logger injected, we'll just proceed.
		_ = s.fileStorage.DeleteImage(ctx, att.FileURL)

		// Delete from DB
		if err := s.attachmentRepo.Delete(ctx, att.ID); err != nil {
			return fmt.Errorf("failed to delete attachment record: %w", err)
		}
	}

	// 5. Delete Thread
	return s.threadRepo.Delete(ctx, threadID)
}
