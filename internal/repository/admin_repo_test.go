package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/repository"
)

func TestAdminRepository_CreateAndGetByID(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewAdminRepository(db)
	ctx := context.Background()

	admin := &models.Admin{ID: "admin-1", Email: "admin1@example.com", Name: "Admin One", PasswordHash: "hash", Role: "admin"}
	require.NoError(t, repo.Create(ctx, admin))

	fetched, err := repo.GetByID(ctx, "admin-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "admin1@example.com", fetched.Email)
}

func TestAdminRepository_GetByEmail(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewAdminRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Admin{
		ID: "admin-1", Email: "admin1@example.com", Name: "Admin One", PasswordHash: "hash", Role: "admin",
	}))

	fetched, err := repo.GetByEmail(ctx, "admin1@example.com")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "admin-1", fetched.ID)

	missing, err := repo.GetByEmail(ctx, "nobody@example.com")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestAdminRepository_UniqueEmailConstraint(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewAdminRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Admin{ID: "admin-1", Email: "dup@example.com", Name: "A", PasswordHash: "h", Role: "admin"}))

	err := repo.Create(ctx, &models.Admin{ID: "admin-2", Email: "dup@example.com", Name: "B", PasswordHash: "h", Role: "admin"})
	assert.Error(t, err)
}

func TestAdminRepository_ListPaginated(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewAdminRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, &models.Admin{
			ID: "admin-" + string(rune('1'+i)), Email: "admin" + string(rune('1'+i)) + "@example.com",
			Name: "Admin", PasswordHash: "h", Role: "admin",
		}))
	}

	page, total, err := repo.List(ctx, 0, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	assert.Len(t, page, 2)
}

func TestAdminRepository_UpdateAndDelete(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewAdminRepository(db)
	ctx := context.Background()

	admin := &models.Admin{ID: "admin-1", Email: "admin1@example.com", Name: "Old Name", PasswordHash: "h", Role: "admin"}
	require.NoError(t, repo.Create(ctx, admin))

	admin.Name = "New Name"
	require.NoError(t, repo.Update(ctx, admin))

	fetched, err := repo.GetByID(ctx, "admin-1")
	require.NoError(t, err)
	assert.Equal(t, "New Name", fetched.Name)

	require.NoError(t, repo.Delete(ctx, "admin-1"))
	fetched, err = repo.GetByID(ctx, "admin-1")
	require.NoError(t, err)
	assert.Nil(t, fetched)
}
