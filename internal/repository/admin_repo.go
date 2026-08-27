package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"tixora/internal/models"
)

type IAdminRepository interface {
	GetByID(ctx context.Context, id string) (*models.Admin, error)
	GetByEmail(ctx context.Context, email string) (*models.Admin, error)
	List(ctx context.Context, offset, limit int) ([]models.Admin, int64, error)
	Create(ctx context.Context, admin *models.Admin) error
	Update(ctx context.Context, admin *models.Admin) error
	Delete(ctx context.Context, id string) error
}

type AdminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) IAdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) GetByID(ctx context.Context, id string) (*models.Admin, error) {
	var admin models.Admin
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get admin by id: %w", err)
	}
	return &admin, nil
}

func (r *AdminRepository) GetByEmail(ctx context.Context, email string) (*models.Admin, error) {
	var admin models.Admin
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get admin by email: %w", err)
	}
	return &admin, nil
}

func (r *AdminRepository) List(ctx context.Context, offset, limit int) ([]models.Admin, int64, error) {
	var admins []models.Admin
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Admin{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count admins: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at desc").
		Find(&admins).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list admins: %w", err)
	}

	return admins, total, nil
}

func (r *AdminRepository) Create(ctx context.Context, admin *models.Admin) error {
	if err := r.db.WithContext(ctx).Create(admin).Error; err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}
	return nil
}

func (r *AdminRepository) Update(ctx context.Context, admin *models.Admin) error {
	if err := r.db.WithContext(ctx).Save(admin).Error; err != nil {
		return fmt.Errorf("failed to update admin: %w", err)
	}
	return nil
}

func (r *AdminRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Admin{}).Error; err != nil {
		return fmt.Errorf("failed to delete admin: %w", err)
	}
	return nil
}
