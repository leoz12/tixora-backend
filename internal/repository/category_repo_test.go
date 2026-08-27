package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/repository"
)

func newTestCategory(id, name, slug string, active bool) *models.Category {
	return &models.Category{ID: id, Name: name, Slug: slug, IsActive: active}
}

func TestCategoryRepository_CreateAndGetByID(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewCategoryRepository(db)
	ctx := context.Background()

	category := newTestCategory("cat-1", "Music", "music", true)
	require.NoError(t, repo.Create(ctx, category))

	fetched, err := repo.GetByID(ctx, "cat-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "Music", fetched.Name)
}

func TestCategoryRepository_GetByID_NotFoundReturnsNilNil(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewCategoryRepository(db)

	fetched, err := repo.GetByID(context.Background(), "does-not-exist")
	require.NoError(t, err)
	assert.Nil(t, fetched)
}

func TestCategoryRepository_GetByNameAndSlug(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewCategoryRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTestCategory("cat-1", "Sports", "sports", true)))

	byName, err := repo.GetByName(ctx, "Sports")
	require.NoError(t, err)
	require.NotNil(t, byName)
	assert.Equal(t, "cat-1", byName.ID)

	bySlug, err := repo.GetBySlug(ctx, "sports")
	require.NoError(t, err)
	require.NotNil(t, bySlug)
	assert.Equal(t, "cat-1", bySlug.ID)
}

func TestCategoryRepository_UniqueNameConstraint(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewCategoryRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTestCategory("cat-1", "Theatre", "theatre", true)))

	err := repo.Create(ctx, newTestCategory("cat-2", "Theatre", "theatre-2", true))
	assert.Error(t, err, "duplicate category name must violate the unique index")
}

func TestCategoryRepository_List_ExcludesInactiveByDefault(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewCategoryRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTestCategory("cat-1", "Active Cat", "active-cat", true)))

	// GORM's Create omits zero-valued fields that carry a `default:` tag
	// (IsActive defaults to true at the DB level), so a category is created
	// active and only deactivated via a subsequent Update - exactly how
	// CategoryService.CreateCategory + UpdateCategory behave in practice.
	inactive := newTestCategory("cat-2", "Inactive Cat", "inactive-cat", true)
	require.NoError(t, repo.Create(ctx, inactive))
	inactive.IsActive = false
	require.NoError(t, repo.Update(ctx, inactive))

	activeOnly, err := repo.List(ctx, false)
	require.NoError(t, err)
	assert.Len(t, activeOnly, 1)
	assert.Equal(t, "cat-1", activeOnly[0].ID)

	all, err := repo.List(ctx, true)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestCategoryRepository_ListWithPagination(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewCategoryRepository(db)
	ctx := context.Background()

	for i, name := range []string{"Alpha", "Beta", "Gamma"} {
		require.NoError(t, repo.Create(ctx, newTestCategory(
			"cat-page-"+string(rune('1'+i)), name, "slug-page-"+string(rune('1'+i)), true,
		)))
	}

	page, total, err := repo.ListWithPagination(ctx, 0, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	assert.Len(t, page, 2)
}

func TestCategoryRepository_Update(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewCategoryRepository(db)
	ctx := context.Background()

	category := newTestCategory("cat-1", "Old Name", "old-name", true)
	require.NoError(t, repo.Create(ctx, category))

	category.Name = "New Name"
	category.IsActive = false
	require.NoError(t, repo.Update(ctx, category))

	fetched, err := repo.GetByID(ctx, "cat-1")
	require.NoError(t, err)
	assert.Equal(t, "New Name", fetched.Name)
	assert.False(t, fetched.IsActive)
}

func TestCategoryRepository_Delete(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewCategoryRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTestCategory("cat-1", "To Delete", "to-delete", true)))
	require.NoError(t, repo.Delete(ctx, "cat-1"))

	fetched, err := repo.GetByID(ctx, "cat-1")
	require.NoError(t, err)
	assert.Nil(t, fetched)
}
