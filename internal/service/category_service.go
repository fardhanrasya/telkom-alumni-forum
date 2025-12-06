package service

import (
	"context"
	"fmt"
	"strings"

	"anoa.com/telkomalumiforum/internal/dto"
	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
)

type CategoryService interface {
	CreateCategory(ctx context.Context, req dto.CreateCategoryRequest) error
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
