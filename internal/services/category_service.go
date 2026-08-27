package services

import (
	"context"
	"fmt"

	"tixora/internal/models"
	"tixora/internal/repository"
	"tixora/internal/utils"
)

const (
	defaultCategoryPageLimit = 20
	maxCategoryPageLimit     = 50
)

// ICategoryService contains event category business logic.
type ICategoryService interface {
	ListCategories(ctx context.Context, includeInactive bool) ([]models.Category, error)
	ListCategoriesWithPagination(ctx context.Context, page, limit int) ([]models.Category, int64, error)
	GetCategoryByID(ctx context.Context, id string) (*models.Category, error)
	CreateCategory(ctx context.Context, name string) (*models.Category, error)
	UpdateCategory(ctx context.Context, id, name string, isActive *bool) (*models.Category, error)
	DeleteCategory(ctx context.Context, id string) error
}

type CategoryService struct {
	repo      repository.ICategoryRepository
	eventRepo repository.IEventRepository
}

func NewCategoryService(repo repository.ICategoryRepository, eventRepo repository.IEventRepository) ICategoryService {
	return &CategoryService{repo: repo, eventRepo: eventRepo}
}

func (s *CategoryService) ListCategories(ctx context.Context, includeInactive bool) ([]models.Category, error) {
	categories, err := s.repo.List(ctx, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	return categories, nil
}

func (s *CategoryService) ListCategoriesWithPagination(ctx context.Context, page, limit int) ([]models.Category, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxCategoryPageLimit {
		limit = defaultCategoryPageLimit
	}

	offset := (page - 1) * limit
	categories, total, err := s.repo.ListWithPagination(ctx, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list categories: %w", err)
	}

	return categories, total, nil
}

func (s *CategoryService) GetCategoryByID(ctx context.Context, id string) (*models.Category, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: category id is required", utils.ErrInvalidInput)
	}

	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch category: %w", err)
	}
	if category == nil {
		return nil, fmt.Errorf("%w: category", utils.ErrNotFound)
	}

	return category, nil
}

func (s *CategoryService) CreateCategory(ctx context.Context, name string) (*models.Category, error) {
	if !utils.ValidateName(name) {
		return nil, fmt.Errorf("%w: name must be between 3 and 255 characters", utils.ErrInvalidInput)
	}

	existing, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing category: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: category with this name already exists", utils.ErrConflict)
	}

	slug := utils.Slugify(name)
	slugMatch, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing category: %w", err)
	}
	if slugMatch != nil {
		return nil, fmt.Errorf("%w: category with a matching slug already exists", utils.ErrConflict)
	}

	category := &models.Category{
		ID:       utils.GenerateUUID(),
		Name:     name,
		Slug:     slug,
		IsActive: true,
	}

	if err := s.repo.Create(ctx, category); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return category, nil
}

func (s *CategoryService) UpdateCategory(ctx context.Context, id, name string, isActive *bool) (*models.Category, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: category id is required", utils.ErrInvalidInput)
	}
	if !utils.ValidateName(name) {
		return nil, fmt.Errorf("%w: name must be between 3 and 255 characters", utils.ErrInvalidInput)
	}

	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch category: %w", err)
	}
	if category == nil {
		return nil, fmt.Errorf("%w: category", utils.ErrNotFound)
	}

	if name != category.Name {
		existing, err := s.repo.GetByName(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing category: %w", err)
		}
		if existing != nil && existing.ID != id {
			return nil, fmt.Errorf("%w: category with this name already exists", utils.ErrConflict)
		}

		slug := utils.Slugify(name)
		slugMatch, err := s.repo.GetBySlug(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing category: %w", err)
		}
		if slugMatch != nil && slugMatch.ID != id {
			return nil, fmt.Errorf("%w: category with a matching slug already exists", utils.ErrConflict)
		}

		category.Name = name
		category.Slug = slug
	}

	if isActive != nil {
		category.IsActive = *isActive
	}

	if err := s.repo.Update(ctx, category); err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	return category, nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: category id is required", utils.ErrInvalidInput)
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to fetch category: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("%w: category", utils.ErrNotFound)
	}

	eventCount, err := s.eventRepo.CountByCategoryID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check category usage: %w", err)
	}
	if eventCount > 0 {
		return fmt.Errorf("%w: category is still used by existing events", utils.ErrConflict)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}
