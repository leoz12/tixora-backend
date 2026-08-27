package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/repository"
)

func TestRefreshTokenRepository_CreateAndGetByHash(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewRefreshTokenRepository(db)
	ctx := context.Background()

	token := &models.RefreshToken{
		ID: "rt-1", SubjectID: "user-1", SubjectType: models.SubjectTypeUser,
		TokenHash: "hash-abc", ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.Create(ctx, token))

	fetched, err := repo.GetByHash(ctx, "hash-abc")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "user-1", fetched.SubjectID)
	assert.Nil(t, fetched.RevokedAt)
}

func TestRefreshTokenRepository_GetByHash_NotFound(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewRefreshTokenRepository(db)

	fetched, err := repo.GetByHash(context.Background(), "does-not-exist")
	require.NoError(t, err)
	assert.Nil(t, fetched)
}

func TestRefreshTokenRepository_UniqueHashConstraint(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewRefreshTokenRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.RefreshToken{
		ID: "rt-1", SubjectID: "user-1", SubjectType: models.SubjectTypeUser,
		TokenHash: "dup-hash", ExpiresAt: time.Now().Add(time.Hour),
	}))

	err := repo.Create(ctx, &models.RefreshToken{
		ID: "rt-2", SubjectID: "user-2", SubjectType: models.SubjectTypeUser,
		TokenHash: "dup-hash", ExpiresAt: time.Now().Add(time.Hour),
	})
	assert.Error(t, err)
}

func TestRefreshTokenRepository_Revoke(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewRefreshTokenRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.RefreshToken{
		ID: "rt-1", SubjectID: "user-1", SubjectType: models.SubjectTypeUser,
		TokenHash: "hash-abc", ExpiresAt: time.Now().Add(time.Hour),
	}))

	replacementID := "rt-2"
	require.NoError(t, repo.Revoke(ctx, "hash-abc", &replacementID))

	fetched, err := repo.GetByHash(ctx, "hash-abc")
	require.NoError(t, err)
	require.NotNil(t, fetched.RevokedAt)
	require.NotNil(t, fetched.ReplacedBy)
	assert.Equal(t, "rt-2", *fetched.ReplacedBy)
}

func TestRefreshTokenRepository_RevokeAllForSubject(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewRefreshTokenRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.RefreshToken{
		ID: "rt-1", SubjectID: "user-1", SubjectType: models.SubjectTypeUser,
		TokenHash: "hash-1", ExpiresAt: time.Now().Add(time.Hour),
	}))
	require.NoError(t, repo.Create(ctx, &models.RefreshToken{
		ID: "rt-2", SubjectID: "user-1", SubjectType: models.SubjectTypeUser,
		TokenHash: "hash-2", ExpiresAt: time.Now().Add(time.Hour),
	}))
	// A token belonging to a different subject must be unaffected.
	require.NoError(t, repo.Create(ctx, &models.RefreshToken{
		ID: "rt-3", SubjectID: "user-2", SubjectType: models.SubjectTypeUser,
		TokenHash: "hash-3", ExpiresAt: time.Now().Add(time.Hour),
	}))

	require.NoError(t, repo.RevokeAllForSubject(ctx, "user-1", models.SubjectTypeUser))

	t1, err := repo.GetByHash(ctx, "hash-1")
	require.NoError(t, err)
	assert.NotNil(t, t1.RevokedAt)

	t2, err := repo.GetByHash(ctx, "hash-2")
	require.NoError(t, err)
	assert.NotNil(t, t2.RevokedAt)

	t3, err := repo.GetByHash(ctx, "hash-3")
	require.NoError(t, err)
	assert.Nil(t, t3.RevokedAt, "a different subject's token must not be revoked")
}
