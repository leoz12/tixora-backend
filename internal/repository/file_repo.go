package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"tixora/internal/models"
)

type IFileRepository interface {
	Create(ctx context.Context, file *models.File) error
	GetByID(ctx context.Context, id string) (*models.File, error)
	Update(ctx context.Context, file *models.File) error
}

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) IFileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(ctx context.Context, file *models.File) error {
	if err := r.db.WithContext(ctx).Create(file).Error; err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return nil
}

func (r *FileRepository) GetByID(ctx context.Context, id string) (*models.File, error) {
	var file models.File
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file by id: %w", err)
	}
	return &file, nil
}

func (r *FileRepository) Update(ctx context.Context, file *models.File) error {
	if err := r.db.WithContext(ctx).Save(file).Error; err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}
	return nil
}
