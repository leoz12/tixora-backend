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

func newCategoryService(repo *mockCategoryRepo, eventRepo *mockEventRepo) services.ICategoryService {
	return services.NewCategoryService(repo, eventRepo)
}

func TestCategoryService_CreateCategory_Success(t *testing.T) {
	repo := new(mockCategoryRepo)
	repo.On("GetByName", mock.Anything, "Music").Return(nil, nil)
	repo.On("GetBySlug", mock.Anything, "music").Return(nil, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*models.Category")).Return(nil)

	svc := newCategoryService(repo, new(mockEventRepo))

	category, err := svc.CreateCategory(context.Background(), "Music")
	require.NoError(t, err)
	assert.Equal(t, "Music", category.Name)
	assert.Equal(t, "music", category.Slug)
	assert.True(t, category.IsActive)
	repo.AssertExpectations(t)
}

func TestCategoryService_CreateCategory_NameTooShort(t *testing.T) {
	svc := newCategoryService(new(mockCategoryRepo), new(mockEventRepo))

	_, err := svc.CreateCategory(context.Background(), "AB")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestCategoryService_CreateCategory_DuplicateName(t *testing.T) {
	repo := new(mockCategoryRepo)
	repo.On("GetByName", mock.Anything, "Music").
		Return(&models.Category{ID: "existing", Name: "Music"}, nil)

	svc := newCategoryService(repo, new(mockEventRepo))

	_, err := svc.CreateCategory(context.Background(), "Music")
	assert.ErrorIs(t, err, utils.ErrConflict)
}

func TestCategoryService_CreateCategory_DuplicateSlug(t *testing.T) {
	repo := new(mockCategoryRepo)
	repo.On("GetByName", mock.Anything, "Music!!").Return(nil, nil)
	repo.On("GetBySlug", mock.Anything, "music").
		Return(&models.Category{ID: "existing", Slug: "music"}, nil)

	svc := newCategoryService(repo, new(mockEventRepo))

	_, err := svc.CreateCategory(context.Background(), "Music!!")
	assert.ErrorIs(t, err, utils.ErrConflict)
}

func TestCategoryService_UpdateCategory_NotFound(t *testing.T) {
	repo := new(mockCategoryRepo)
	repo.On("GetByID", mock.Anything, "missing").Return(nil, nil)

	svc := newCategoryService(repo, new(mockEventRepo))

	_, err := svc.UpdateCategory(context.Background(), "missing", "New Name", nil)
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestCategoryService_UpdateCategory_SameNameSkipsUniquenessCheck(t *testing.T) {
	existing := &models.Category{ID: "cat-1", Name: "Music", Slug: "music", IsActive: true}
	repo := new(mockCategoryRepo)
	repo.On("GetByID", mock.Anything, "cat-1").Return(existing, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*models.Category")).Return(nil)

	svc := newCategoryService(repo, new(mockEventRepo))

	isActive := false
	updated, err := svc.UpdateCategory(context.Background(), "cat-1", "Music", &isActive)
	require.NoError(t, err)
	assert.False(t, updated.IsActive)
	repo.AssertNotCalled(t, "GetByName", mock.Anything, mock.Anything)
}

func TestCategoryService_UpdateCategory_NameConflictWithAnotherCategory(t *testing.T) {
	existing := &models.Category{ID: "cat-1", Name: "Music", Slug: "music"}
	repo := new(mockCategoryRepo)
	repo.On("GetByID", mock.Anything, "cat-1").Return(existing, nil)
	repo.On("GetByName", mock.Anything, "Sports").
		Return(&models.Category{ID: "cat-2", Name: "Sports"}, nil)

	svc := newCategoryService(repo, new(mockEventRepo))

	_, err := svc.UpdateCategory(context.Background(), "cat-1", "Sports", nil)
	assert.ErrorIs(t, err, utils.ErrConflict)
}

func TestCategoryService_DeleteCategory_NotFound(t *testing.T) {
	repo := new(mockCategoryRepo)
	repo.On("GetByID", mock.Anything, "missing").Return(nil, nil)

	svc := newCategoryService(repo, new(mockEventRepo))

	err := svc.DeleteCategory(context.Background(), "missing")
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestCategoryService_DeleteCategory_RejectedWhenInUse(t *testing.T) {
	repo := new(mockCategoryRepo)
	repo.On("GetByID", mock.Anything, "cat-1").Return(&models.Category{ID: "cat-1"}, nil)
	eventRepo := new(mockEventRepo)
	eventRepo.On("CountByCategoryID", mock.Anything, "cat-1").Return(3, nil)

	svc := newCategoryService(repo, eventRepo)

	err := svc.DeleteCategory(context.Background(), "cat-1")
	assert.ErrorIs(t, err, utils.ErrConflict)
}

func TestCategoryService_DeleteCategory_Success(t *testing.T) {
	repo := new(mockCategoryRepo)
	repo.On("GetByID", mock.Anything, "cat-1").Return(&models.Category{ID: "cat-1"}, nil)
	repo.On("Delete", mock.Anything, "cat-1").Return(nil)
	eventRepo := new(mockEventRepo)
	eventRepo.On("CountByCategoryID", mock.Anything, "cat-1").Return(0, nil)

	svc := newCategoryService(repo, eventRepo)

	err := svc.DeleteCategory(context.Background(), "cat-1")
	require.NoError(t, err)
	repo.AssertExpectations(t)
}
