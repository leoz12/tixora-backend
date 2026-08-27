package services

import (
	"context"
	"fmt"
	"time"

	"tixora/internal/config"
	"tixora/internal/models"
	"tixora/internal/repository"
	"tixora/internal/utils"
)

const (
	defaultAdminRole      = "admin"
	defaultAdminPageLimit = 20
	maxAdminPageLimit     = 50
	minAdminPasswordLen   = 8
)

// IAdminService contains admin authentication and management business logic.
type IAdminService interface {
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, admin *models.Admin, err error)
	RefreshToken(ctx context.Context, rawRefreshToken string) (accessToken, refreshToken string, admin *models.Admin, err error)
	Logout(ctx context.Context, rawRefreshToken string) error
	GetAdminByID(ctx context.Context, id string) (*models.Admin, error)
	ListAdmins(ctx context.Context, page, limit int) ([]models.Admin, int64, error)
	EnsureBootstrapAdmin(ctx context.Context, email, name, password string) error
	CreateAdmin(ctx context.Context, email, name, password, role string) (*models.Admin, error)
	UpdateAdmin(ctx context.Context, id, name, password, role string) (*models.Admin, error)
	DeleteAdmin(ctx context.Context, id string) error
}

type AdminService struct {
	repo             repository.IAdminRepository
	refreshTokenRepo repository.IRefreshTokenRepository
	cfg              *config.Config
}

func NewAdminService(repo repository.IAdminRepository, refreshTokenRepo repository.IRefreshTokenRepository, cfg *config.Config) IAdminService {
	return &AdminService{repo: repo, refreshTokenRepo: refreshTokenRepo, cfg: cfg}
}

func (s *AdminService) Login(ctx context.Context, email, password string) (string, string, *models.Admin, error) {
	if !utils.ValidateEmail(email) {
		return "", "", nil, fmt.Errorf("%w: invalid email", utils.ErrInvalidInput)
	}
	if password == "" {
		return "", "", nil, fmt.Errorf("%w: password is required", utils.ErrInvalidInput)
	}

	admin, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to fetch admin: %w", err)
	}
	if admin == nil || !utils.CheckPassword(password, admin.PasswordHash) {
		return "", "", nil, fmt.Errorf("%w: invalid email or password", utils.ErrUnauthorized)
	}

	accessToken, rawRefreshToken, _, err := s.issueTokens(ctx, admin)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, rawRefreshToken, admin, nil
}

// RefreshToken validates a raw refresh token against the stored hash,
// rotates it, and returns a new access/refresh pair. Reuse of an
// already-revoked token revokes every refresh token for that admin.
func (s *AdminService) RefreshToken(ctx context.Context, rawRefreshToken string) (string, string, *models.Admin, error) {
	if rawRefreshToken == "" {
		return "", "", nil, fmt.Errorf("%w: refresh token is required", utils.ErrUnauthorized)
	}

	tokenHash := utils.HashToken(rawRefreshToken)
	stored, err := s.refreshTokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to look up refresh token: %w", err)
	}
	if stored == nil {
		return "", "", nil, fmt.Errorf("%w: invalid refresh token", utils.ErrUnauthorized)
	}

	if stored.RevokedAt != nil {
		if revokeErr := s.refreshTokenRepo.RevokeAllForSubject(ctx, stored.SubjectID, stored.SubjectType); revokeErr != nil {
			return "", "", nil, fmt.Errorf("failed to revoke compromised sessions: %w", revokeErr)
		}
		return "", "", nil, fmt.Errorf("%w: refresh token reuse detected", utils.ErrUnauthorized)
	}

	if stored.ExpiresAt.Before(time.Now()) {
		return "", "", nil, fmt.Errorf("%w: refresh token expired", utils.ErrUnauthorized)
	}

	admin, err := s.repo.GetByID(ctx, stored.SubjectID)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to fetch admin: %w", err)
	}
	if admin == nil {
		return "", "", nil, fmt.Errorf("%w: admin", utils.ErrNotFound)
	}

	newAccessToken, newRawRefreshToken, newRowID, err := s.issueTokens(ctx, admin)
	if err != nil {
		return "", "", nil, err
	}

	if err := s.refreshTokenRepo.Revoke(ctx, tokenHash, &newRowID); err != nil {
		return "", "", nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	return newAccessToken, newRawRefreshToken, admin, nil
}

// Logout revokes the given refresh token. Idempotent - logging out with an
// empty or already-revoked token is not an error.
func (s *AdminService) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}

	if err := s.refreshTokenRepo.Revoke(ctx, utils.HashToken(rawRefreshToken), nil); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// issueTokens generates a fresh access JWT and a fresh (persisted, hashed)
// refresh token for the given admin, returning the new refresh token row's
// ID alongside them so callers can link a rotated-out token to its
// replacement.
func (s *AdminService) issueTokens(ctx context.Context, admin *models.Admin) (string, string, string, error) {
	accessToken, err := utils.GenerateAdminJWT(admin.ID, admin.Email, admin.Role, s.cfg.JWTSecret, s.cfg.JWTAccessExpiry)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	rawRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshRow := &models.RefreshToken{
		ID:          utils.GenerateUUID(),
		SubjectID:   admin.ID,
		SubjectType: models.SubjectTypeAdmin,
		TokenHash:   utils.HashToken(rawRefreshToken),
		ExpiresAt:   time.Now().Add(s.cfg.JWTRefreshExpiry),
	}
	if err := s.refreshTokenRepo.Create(ctx, refreshRow); err != nil {
		return "", "", "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return accessToken, rawRefreshToken, refreshRow.ID, nil
}

func (s *AdminService) GetAdminByID(ctx context.Context, id string) (*models.Admin, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: admin id is required", utils.ErrInvalidInput)
	}

	admin, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch admin: %w", err)
	}
	if admin == nil {
		return nil, fmt.Errorf("%w: admin", utils.ErrNotFound)
	}

	return admin, nil
}

func (s *AdminService) ListAdmins(ctx context.Context, page, limit int) ([]models.Admin, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxAdminPageLimit {
		limit = defaultAdminPageLimit
	}

	offset := (page - 1) * limit
	admins, total, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list admins: %w", err)
	}

	return admins, total, nil
}

// EnsureBootstrapAdmin creates an initial admin account from configuration on
// a fresh deployment, since every admin-management route requires an existing
// admin session and there's no other way to create the first one.
//
// It's opt-in: a no-op when both email and password are empty. It's also
// idempotent - once an admin with that email exists, it does nothing - so it's
// safe to leave the env vars set across redeploys.
func (s *AdminService) EnsureBootstrapAdmin(ctx context.Context, email, name, password string) error {
	if email == "" && password == "" {
		return nil
	}
	if email == "" || password == "" {
		return fmt.Errorf("%w: admin bootstrap requires both email and password", utils.ErrInvalidInput)
	}

	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to check for existing bootstrap admin: %w", err)
	}
	if existing != nil {
		return nil
	}

	if name == "" {
		name = "Administrator"
	}

	if _, err := s.CreateAdmin(ctx, email, name, password, defaultAdminRole); err != nil {
		return fmt.Errorf("failed to create bootstrap admin: %w", err)
	}

	return nil
}

func (s *AdminService) CreateAdmin(ctx context.Context, email, name, password, role string) (*models.Admin, error) {
	if !utils.ValidateEmail(email) {
		return nil, fmt.Errorf("%w: invalid email", utils.ErrInvalidInput)
	}
	if !utils.ValidateName(name) {
		return nil, fmt.Errorf("%w: name must be between 3 and 255 characters", utils.ErrInvalidInput)
	}
	if len(password) < minAdminPasswordLen {
		return nil, fmt.Errorf("%w: password must be at least %d characters", utils.ErrInvalidInput, minAdminPasswordLen)
	}

	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing admin: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: admin with this email already exists", utils.ErrConflict)
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	if role == "" {
		role = defaultAdminRole
	}

	admin := &models.Admin{
		ID:           utils.GenerateUUID(),
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		Role:         role,
	}

	if err := s.repo.Create(ctx, admin); err != nil {
		return nil, fmt.Errorf("failed to create admin: %w", err)
	}

	return admin, nil
}

func (s *AdminService) UpdateAdmin(ctx context.Context, id, name, password, role string) (*models.Admin, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: admin id is required", utils.ErrInvalidInput)
	}
	if !utils.ValidateName(name) {
		return nil, fmt.Errorf("%w: name must be between 3 and 255 characters", utils.ErrInvalidInput)
	}

	admin, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch admin: %w", err)
	}
	if admin == nil {
		return nil, fmt.Errorf("%w: admin", utils.ErrNotFound)
	}

	admin.Name = name
	if role != "" {
		admin.Role = role
	}
	if password != "" {
		if len(password) < minAdminPasswordLen {
			return nil, fmt.Errorf("%w: password must be at least %d characters", utils.ErrInvalidInput, minAdminPasswordLen)
		}
		passwordHash, err := utils.HashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		admin.PasswordHash = passwordHash
	}

	if err := s.repo.Update(ctx, admin); err != nil {
		return nil, fmt.Errorf("failed to update admin: %w", err)
	}

	return admin, nil
}

func (s *AdminService) DeleteAdmin(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: admin id is required", utils.ErrInvalidInput)
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to fetch admin: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("%w: admin", utils.ErrNotFound)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete admin: %w", err)
	}

	return nil
}
