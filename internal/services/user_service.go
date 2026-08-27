package services

import (
	"context"
	"fmt"

	"tixora/internal/models"
	"tixora/internal/repository"
	"tixora/internal/utils"
)

const (
	defaultUserPageLimit = 20
	maxUserPageLimit     = 50
)

// IUserService contains user profile business logic.
type IUserService interface {
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateProfile(ctx context.Context, id, name, avatarURL string) (*models.User, error)
	ListUsers(ctx context.Context, page, limit int, search string) ([]models.User, int64, error)
}

type UserService struct {
	repo repository.IUserRepository
}

func NewUserService(repo repository.IUserRepository) IUserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: user id is required", utils.ErrInvalidInput)
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("%w: user", utils.ErrNotFound)
	}

	return user, nil
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if !utils.ValidateEmail(email) {
		return nil, fmt.Errorf("%w: invalid email", utils.ErrInvalidInput)
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("%w: user", utils.ErrNotFound)
	}

	return user, nil
}

// ListUsers returns a paginated list of users, optionally filtered by a
// search term matched against name and email, for admin use.
func (s *UserService) ListUsers(ctx context.Context, page, limit int, search string) ([]models.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxUserPageLimit {
		limit = defaultUserPageLimit
	}

	offset := (page - 1) * limit
	users, total, err := s.repo.List(ctx, search, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch users: %w", err)
	}

	return users, total, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, id, name, avatarURL string) (*models.User, error) {
	if !utils.ValidateName(name) {
		return nil, fmt.Errorf("%w: name must be between 3 and 255 characters", utils.ErrInvalidInput)
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("%w: user", utils.ErrNotFound)
	}

	user.Name = name
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}
