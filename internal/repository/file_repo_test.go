package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/repository"
)

func TestFileRepository_CreateAndGetByID(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewFileRepository(db)
	ctx := context.Background()

	file := &models.File{
		ID: "file-1", ObjectKey: "uploads/file-1.jpg", OriginalName: "cover.jpg",
		Provider: "r2", MimeType: "image/jpeg", Size: 1024, Status: models.FileStatusPending,
	}
	require.NoError(t, repo.Create(ctx, file))

	fetched, err := repo.GetByID(ctx, "file-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, models.FileStatusPending, fetched.Status)
}

func TestFileRepository_GetByID_NotFound(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewFileRepository(db)

	fetched, err := repo.GetByID(context.Background(), "missing")
	require.NoError(t, err)
	assert.Nil(t, fetched)
}

func TestFileRepository_UniqueObjectKeyConstraint(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewFileRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.File{
		ID: "file-1", ObjectKey: "uploads/dup.jpg", MimeType: "image/jpeg", Size: 1, Status: models.FileStatusPending,
	}))

	err := repo.Create(ctx, &models.File{
		ID: "file-2", ObjectKey: "uploads/dup.jpg", MimeType: "image/jpeg", Size: 1, Status: models.FileStatusPending,
	})
	assert.Error(t, err)
}

func TestFileRepository_Update(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewFileRepository(db)
	ctx := context.Background()

	file := &models.File{
		ID: "file-1", ObjectKey: "uploads/file-1.jpg", MimeType: "image/jpeg", Size: 1024, Status: models.FileStatusPending,
	}
	require.NoError(t, repo.Create(ctx, file))

	file.Status = models.FileStatusAttached
	require.NoError(t, repo.Update(ctx, file))

	fetched, err := repo.GetByID(ctx, "file-1")
	require.NoError(t, err)
	assert.Equal(t, models.FileStatusAttached, fetched.Status)
}
