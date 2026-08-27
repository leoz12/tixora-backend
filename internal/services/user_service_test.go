package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/services"
	"tixora/internal/utils"
)

func newUserService(repo *mockUserRepo) services.IUserService {
	return services.NewUserService(repo)
}

func TestUserService_GetUserByID_EmptyIDRejected(t *testing.T) {
	svc := newUserService(new(mockUserRepo))

	_, err := svc.GetUserByID(context.Background(), "")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	repo.On("GetByID", mock.Anything, "missing").Return(nil, nil)

	svc := newUserService(repo)

	_, err := svc.GetUserByID(context.Background(), "missing")
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestUserService_GetUserByEmail_InvalidEmailRejected(t *testing.T) {
	svc := newUserService(new(mockUserRepo))

	_, err := svc.GetUserByEmail(context.Background(), "not-an-email")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestUserService_UpdateProfile_InvalidNameRejected(t *testing.T) {
	svc := newUserService(new(mockUserRepo))

	_, err := svc.UpdateProfile(context.Background(), "user-1", "A", "")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestUserService_UpdateProfile_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	repo.On("GetByID", mock.Anything, "user-1").Return(nil, nil)

	svc := newUserService(repo)

	_, err := svc.UpdateProfile(context.Background(), "user-1", "New Name", "")
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestUserService_UpdateProfile_KeepsAvatarWhenEmpty(t *testing.T) {
	existing := &models.User{ID: "user-1", Name: "Old Name", AvatarURL: "https://cdn/avatar.png"}
	repo := new(mockUserRepo)
	repo.On("GetByID", mock.Anything, "user-1").Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

	svc := newUserService(repo)

	updated, err := svc.UpdateProfile(context.Background(), "user-1", "New Name", "")
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "https://cdn/avatar.png", updated.AvatarURL)
}

func TestUserService_UpdateProfile_ReplacesAvatarWhenProvided(t *testing.T) {
	existing := &models.User{ID: "user-1", Name: "Old Name", AvatarURL: "https://cdn/old.png"}
	repo := new(mockUserRepo)
	repo.On("GetByID", mock.Anything, "user-1").Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

	svc := newUserService(repo)

	updated, err := svc.UpdateProfile(context.Background(), "user-1", "New Name", "https://cdn/new.png")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn/new.png", updated.AvatarURL)
}

func TestUserService_ListUsers_DefaultsInvalidPagination(t *testing.T) {
	repo := new(mockUserRepo)
	repo.On("List", mock.Anything, "", 0, 20).Return([]models.User{}, 0, nil)

	svc := newUserService(repo)

	_, _, err := svc.ListUsers(context.Background(), 0, 0, "")
	require.NoError(t, err)
	repo.AssertExpectations(t)
}
