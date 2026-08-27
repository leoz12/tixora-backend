package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"tixora/internal/models"
)

type ICategoryRepository interface {
	List(ctx context.Context, includeInactive bool) ([]models.Category, error)
	ListWithPagination(ctx context.Context, offset, limit int) ([]models.Category, int64, error)
	GetByID(ctx context.Context, id string) (*models.Category, error)
	GetByName(ctx context.Context, name string) (*models.Category, error)
	GetBySlug(ctx context.Context, slug string) (*models.Category, error)
	Create(ctx context.Context, category *models.Category) error
	Update(ctx context.Context, category *models.Category) error
	Delete(ctx context.Context, id string) error
}

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) ICategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) List(ctx context.Context, includeInactive bool) ([]models.Category, error) {
	var categories []models.Category

	query := r.db.WithContext(ctx)
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Order("name asc").Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	return categories, nil
}

func (r *CategoryRepository) ListWithPagination(ctx context.Context, offset, limit int) ([]models.Category, int64, error) {
	var categories []models.Category
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Category{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count categories: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("name asc").
		Find(&categories).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list categories: %w", err)
	}

	return categories, total, nil
}

func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*models.Category, error) {
	var category models.Category
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get category by id: %w", err)
	}
	return &category, nil
}

func (r *CategoryRepository) GetByName(ctx context.Context, name string) (*models.Category, error) {
	var category models.Category
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get category by name: %w", err)
	}
	return &category, nil
}

func (r *CategoryRepository) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	var category models.Category
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get category by slug: %w", err)
	}
	return &category, nil
}

func (r *CategoryRepository) Create(ctx context.Context, category *models.Category) error {
	if err := r.db.WithContext(ctx).Create(category).Error; err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) Update(ctx context.Context, category *models.Category) error {
	if err := r.db.WithContext(ctx).Save(category).Error; err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Category{}).Error; err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	return nil
}
