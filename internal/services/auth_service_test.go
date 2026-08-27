package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/services"
	"tixora/internal/utils"
)

// HandleGoogleCallback talks to Google's real token/userinfo endpoints and
// isn't covered here (no injectable HTTP transport) - only the fully local
// paths (input validation, refresh rotation, logout) are unit tested.
// The Google OAuth exchange itself is exercised manually / in a live env.

func newAuthService(userRepo *mockUserRepo, refreshRepo *mockRefreshTokenRepo) services.IAuthService {
	return services.NewAuthService(userRepo, refreshRepo, testConfig())
}

func TestAuthService_HandleGoogleCallback_EmptyCodeRejected(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockRefreshTokenRepo))

	_, _, _, err := svc.HandleGoogleCallback(context.Background(), "")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestAuthService_RefreshToken_EmptyTokenRejected(t *testing.T) {
	svc := newAuthService(new(mockUserRepo), new(mockRefreshTokenRepo))

	_, _, _, err := svc.RefreshToken(context.Background(), "")
	assert.ErrorIs(t, err, utils.ErrUnauthorized)
}

func TestAuthService_RefreshToken_UnknownTokenRejected(t *testing.T) {
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)

	svc := newAuthService(new(mockUserRepo), refreshRepo)

	_, _, _, err := svc.RefreshToken(context.Background(), "raw-token")
	assert.ErrorIs(t, err, utils.ErrUnauthorized)
}

func TestAuthService_RefreshToken_ReusedRevokedTokenRevokesAllSessions(t *testing.T) {
	revokedAt := time.Now().Add(-time.Minute)
	stored := &models.RefreshToken{
		SubjectID: "user-1", SubjectType: models.SubjectTypeUser,
		RevokedAt: &revokedAt, ExpiresAt: time.Now().Add(time.Hour),
	}
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).Return(stored, nil)
	refreshRepo.On("RevokeAllForSubject", mock.Anything, "user-1", models.SubjectTypeUser).Return(nil)

	svc := newAuthService(new(mockUserRepo), refreshRepo)

	_, _, _, err := svc.RefreshToken(context.Background(), "raw-token")
	assert.ErrorIs(t, err, utils.ErrUnauthorized)
	refreshRepo.AssertCalled(t, "RevokeAllForSubject", mock.Anything, "user-1", models.SubjectTypeUser)
}

func TestAuthService_RefreshToken_ExpiredTokenRejected(t *testing.T) {
	stored := &models.RefreshToken{
		SubjectID: "user-1", SubjectType: models.SubjectTypeUser,
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).Return(stored, nil)

	svc := newAuthService(new(mockUserRepo), refreshRepo)

	_, _, _, err := svc.RefreshToken(context.Background(), "raw-token")
	assert.ErrorIs(t, err, utils.ErrUnauthorized)
}

func TestAuthService_RefreshToken_Success(t *testing.T) {
	stored := &models.RefreshToken{
		SubjectID: "user-1", SubjectType: models.SubjectTypeUser,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).Return(stored, nil)
	refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.RefreshToken")).Return(nil)
	refreshRepo.On("Revoke", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*string")).Return(nil)

	userRepo := new(mockUserRepo)
	userRepo.On("GetByID", mock.Anything, "user-1").
		Return(&models.User{ID: "user-1", Email: "user@example.com"}, nil)

	svc := newAuthService(userRepo, refreshRepo)

	accessToken, refreshToken, user, err := svc.RefreshToken(context.Background(), "raw-token")
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.Equal(t, "user-1", user.ID)
	refreshRepo.AssertExpectations(t)
}

func TestAuthService_RefreshToken_UserDeletedAfterTokenIssued(t *testing.T) {
	stored := &models.RefreshToken{
		SubjectID: "user-1", SubjectType: models.SubjectTypeUser,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).Return(stored, nil)

	userRepo := new(mockUserRepo)
	userRepo.On("GetByID", mock.Anything, "user-1").Return(nil, nil)

	svc := newAuthService(userRepo, refreshRepo)

	_, _, _, err := svc.RefreshToken(context.Background(), "raw-token")
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestAuthService_Logout_EmptyTokenIsNoop(t *testing.T) {
	refreshRepo := new(mockRefreshTokenRepo)
	svc := newAuthService(new(mockUserRepo), refreshRepo)

	err := svc.Logout(context.Background(), "")
	require.NoError(t, err)
	refreshRepo.AssertNotCalled(t, "Revoke", mock.Anything, mock.Anything, mock.Anything)
}

func TestAuthService_Logout_RevokesToken(t *testing.T) {
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("Revoke", mock.Anything, mock.AnythingOfType("string"), (*string)(nil)).Return(nil)

	svc := newAuthService(new(mockUserRepo), refreshRepo)

	err := svc.Logout(context.Background(), "raw-token")
	require.NoError(t, err)
	refreshRepo.AssertExpectations(t)
}
