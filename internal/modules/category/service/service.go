package category

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"anoa.com/telkomalumiforum/internal/entity"
	"anoa.com/telkomalumiforum/internal/modules/category/dto"
	"anoa.com/telkomalumiforum/internal/modules/category/repository"
	commonDto "anoa.com/telkomalumiforum/pkg/dto"
)

type CategoryService interface {
	CreateCategory(ctx context.Context, req dto.CreateCategoryRequest) error
	GetAllCategories(ctx context.Context, filter commonDto.CategoryFilter) (*commonDto.PaginatedCategoryResponse, error)
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

	category := &entity.Category{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
	}

	return s.repo.Create(ctx, category)
}

func (s *categoryService) GetAllCategories(ctx context.Context, filter commonDto.CategoryFilter) (*commonDto.PaginatedCategoryResponse, error) {
	categories, err := s.repo.FindAll(ctx, filter.Search)
	if err != nil {
		return nil, err
	}

	var categoryResponses []commonDto.CategoryResponse
	for _, cat := range categories {
		categoryResponses = append(categoryResponses, commonDto.CategoryResponse{
			ID:          cat.ID,
			Name:        cat.Name,
			Slug:        cat.Slug,
			Description: cat.Description,
		})
	}

	return &commonDto.PaginatedCategoryResponse{
		Data: categoryResponses,
		Meta: commonDto.PaginationMeta{
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
