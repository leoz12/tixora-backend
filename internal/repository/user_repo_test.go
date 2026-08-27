package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/repository"
)

func TestUserRepository_CreateAndGetByID(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{ID: "user-1", Email: "user1@example.com", Name: "User One", GoogleID: "google-1"}
	require.NoError(t, repo.Create(ctx, user))

	fetched, err := repo.GetByID(ctx, "user-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "user1@example.com", fetched.Email)
}

func TestUserRepository_GetByEmailAndGoogleID(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.User{
		ID: "user-1", Email: "user1@example.com", Name: "User One", GoogleID: "google-1",
	}))

	byEmail, err := repo.GetByEmail(ctx, "user1@example.com")
	require.NoError(t, err)
	require.NotNil(t, byEmail)
	assert.Equal(t, "user-1", byEmail.ID)

	byGoogleID, err := repo.GetByGoogleID(ctx, "google-1")
	require.NoError(t, err)
	require.NotNil(t, byGoogleID)
	assert.Equal(t, "user-1", byGoogleID.ID)
}

func TestUserRepository_UniqueEmailConstraint(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.User{ID: "user-1", Email: "dup@example.com", Name: "A", GoogleID: "g-1"}))

	err := repo.Create(ctx, &models.User{ID: "user-2", Email: "dup@example.com", Name: "B", GoogleID: "g-2"})
	assert.Error(t, err, "duplicate email must violate the unique index")
}

func TestUserRepository_List_SearchMatchesNameOrEmail(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.User{ID: "user-1", Email: "alice@example.com", Name: "Alice", GoogleID: "g-1"}))
	require.NoError(t, repo.Create(ctx, &models.User{ID: "user-2", Email: "bob@example.com", Name: "Bob", GoogleID: "g-2"}))

	byName, total, err := repo.List(ctx, "Alice", 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, byName, 1)
	assert.Equal(t, "user-1", byName[0].ID)

	all, total, err := repo.List(ctx, "", 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, all, 2)
}

func TestUserRepository_Update(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{ID: "user-1", Email: "user1@example.com", Name: "Old Name", GoogleID: "g-1"}
	require.NoError(t, repo.Create(ctx, user))

	user.Name = "New Name"
	require.NoError(t, repo.Update(ctx, user))

	fetched, err := repo.GetByID(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "New Name", fetched.Name)
}
