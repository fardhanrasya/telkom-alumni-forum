package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"anoa.com/telkomalumiforum/internal/dto"
	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
)

type CategoryService interface {
	CreateCategory(ctx context.Context, req dto.CreateCategoryRequest) error
	GetAllCategories(ctx context.Context, filter dto.CategoryFilter) (*dto.PaginatedCategoryResponse, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}

type categoryService struct {
	repo repository.CategoryRepository
}

func NewCategoryService(repo repository.CategoryRepository) CategoryService {
	return &categoryService{repo: repo}
}

func (s *categoryService) CreateCategory(ctx context.Context, req dto.CreateCategoryRequest) error {
	slug := strings.ReplaceAll(strings.ToLower(req.Name), " ", "-")

	existing, _ := s.repo.FindBySlug(ctx, slug)
	if existing != nil {
		return fmt.Errorf("category with name %s already exists", req.Name)
	}

	category := &model.Category{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
	}

	return s.repo.Create(ctx, category)
}

func (s *categoryService) GetAllCategories(ctx context.Context, filter dto.CategoryFilter) (*dto.PaginatedCategoryResponse, error) {
	categories, err := s.repo.FindAll(ctx, filter.Search)
	if err != nil {
		return nil, err
	}

	var categoryResponses []dto.CategoryResponse
	for _, cat := range categories {
		categoryResponses = append(categoryResponses, dto.CategoryResponse{
			ID:          cat.ID,
			Name:        cat.Name,
			Slug:        cat.Slug,
			Description: cat.Description,
		})
	}

	return &dto.PaginatedCategoryResponse{
		Data: categoryResponses,
		Meta: dto.PaginationMeta{
			TotalItems: int64(len(categories)),
		},
	}, nil
}

func (s *categoryService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("category not found")
	}

	return s.repo.Delete(ctx, id)
}
