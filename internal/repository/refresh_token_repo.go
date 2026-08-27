package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"tixora/internal/models"
)

type IRefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string, replacedBy *string) error
	RevokeAllForSubject(ctx context.Context, subjectID string, subjectType models.SubjectType) error
}

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) IRefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get refresh token by hash: %w", err)
	}
	return &token, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenHash string, replacedBy *string) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Updates(map[string]interface{}{"revoked_at": now, "replaced_by": replacedBy}).Error; err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllForSubject(ctx context.Context, subjectID string, subjectType models.SubjectType) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("subject_id = ? AND subject_type = ? AND revoked_at IS NULL", subjectID, subjectType).
		Update("revoked_at", now).Error; err != nil {
		return fmt.Errorf("failed to revoke refresh tokens for subject: %w", err)
	}
	return nil
}
