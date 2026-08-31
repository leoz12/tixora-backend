package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"tixora/internal/config"
	"tixora/internal/models"
	"tixora/internal/services"
	"tixora/internal/utils"
)

const testJWTSecret = "test_secret_min_32_chars_long_ok"

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:        testJWTSecret,
		JWTAccessExpiry:  15 * time.Minute,
		JWTRefreshExpiry: 168 * time.Hour,
	}
}

func newAdminService(repo *mockAdminRepo, refreshRepo *mockRefreshTokenRepo) services.IAdminService {
	return services.NewAdminService(repo, refreshRepo, testConfig())
}

func TestAdminService_Login_InvalidEmailRejected(t *testing.T) {
	svc := newAdminService(new(mockAdminRepo), new(mockRefreshTokenRepo))

	_, _, _, err := svc.Login(context.Background(), "not-an-email", "password123")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestAdminService_Login_UnknownEmailRejected(t *testing.T) {
	repo := new(mockAdminRepo)
	repo.On("GetByEmail", mock.Anything, "admin@example.com").Return(nil, nil)

	svc := newAdminService(repo, new(mockRefreshTokenRepo))

	_, _, _, err := svc.Login(context.Background(), "admin@example.com", "password123")
	assert.ErrorIs(t, err, utils.ErrUnauthorized)
}

func TestAdminService_Login_WrongPasswordRejected(t *testing.T) {
	hash, err := utils.HashPassword("correct-password")
	require.NoError(t, err)

	repo := new(mockAdminRepo)
	repo.On("GetByEmail", mock.Anything, "admin@example.com").
		Return(&models.Admin{ID: "admin-1", Email: "admin@example.com", PasswordHash: hash}, nil)

	svc := newAdminService(repo, new(mockRefreshTokenRepo))

	_, _, _, err = svc.Login(context.Background(), "admin@example.com", "wrong-password")
	assert.ErrorIs(t, err, utils.ErrUnauthorized)
}

func TestAdminService_Login_Success(t *testing.T) {
	hash, err := utils.HashPassword("correct-password")
	require.NoError(t, err)

	repo := new(mockAdminRepo)
	repo.On("GetByEmail", mock.Anything, "admin@example.com").
		Return(&models.Admin{ID: "admin-1", Email: "admin@example.com", PasswordHash: hash, Role: "admin"}, nil)

	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.RefreshToken")).Return(nil)

	svc := newAdminService(repo, refreshRepo)

	accessToken, refreshToken, admin, err := svc.Login(context.Background(), "admin@example.com", "correct-password")
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.Equal(t, "admin-1", admin.ID)
}

func TestAdminService_RefreshToken_UnknownTokenRejected(t *testing.T) {
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)

	svc := newAdminService(new(mockAdminRepo), refreshRepo)

	_, _, _, err := svc.RefreshToken(context.Background(), "raw-token")
	assert.ErrorIs(t, err, utils.ErrUnauthorized)
}

func TestAdminService_RefreshToken_ReusedRevokedTokenRevokesAllSessions(t *testing.T) {
	revokedAt := time.Now().Add(-time.Minute)
	stored := &models.RefreshToken{
		SubjectID: "admin-1", SubjectType: models.SubjectTypeAdmin,
		RevokedAt: &revokedAt, ExpiresAt: time.Now().Add(time.Hour),
	}
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).Return(stored, nil)
	refreshRepo.On("RevokeAllForSubject", mock.Anything, "admin-1", models.SubjectTypeAdmin).Return(nil)

	svc := newAdminService(new(mockAdminRepo), refreshRepo)

	_, _, _, err := svc.RefreshToken(context.Background(), "raw-token")
	assert.ErrorIs(t, err, utils.ErrUnauthorized)
	refreshRepo.AssertCalled(t, "RevokeAllForSubject", mock.Anything, "admin-1", models.SubjectTypeAdmin)
}

func TestAdminService_RefreshToken_ExpiredTokenRejected(t *testing.T) {
	stored := &models.RefreshToken{
		SubjectID: "admin-1", SubjectType: models.SubjectTypeAdmin,
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).Return(stored, nil)

	svc := newAdminService(new(mockAdminRepo), refreshRepo)

	_, _, _, err := svc.RefreshToken(context.Background(), "raw-token")
	assert.ErrorIs(t, err, utils.ErrUnauthorized)
}

func TestAdminService_RefreshToken_Success(t *testing.T) {
	stored := &models.RefreshToken{
		SubjectID: "admin-1", SubjectType: models.SubjectTypeAdmin,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).Return(stored, nil)
	refreshRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.RefreshToken")).Return(nil)
	refreshRepo.On("Revoke", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*string")).Return(nil)

	adminRepo := new(mockAdminRepo)
	adminRepo.On("GetByID", mock.Anything, "admin-1").
		Return(&models.Admin{ID: "admin-1", Email: "admin@example.com", Role: "admin"}, nil)

	svc := newAdminService(adminRepo, refreshRepo)

	accessToken, refreshToken, admin, err := svc.RefreshToken(context.Background(), "raw-token")
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.Equal(t, "admin-1", admin.ID)
}

func TestAdminService_CreateAdmin_ShortPasswordRejected(t *testing.T) {
	svc := newAdminService(new(mockAdminRepo), new(mockRefreshTokenRepo))

	_, err := svc.CreateAdmin(context.Background(), "new@example.com", "New Admin", "short", "admin")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestAdminService_CreateAdmin_DuplicateEmailRejected(t *testing.T) {
	repo := new(mockAdminRepo)
	repo.On("GetByEmail", mock.Anything, "existing@example.com").
		Return(&models.Admin{ID: "admin-1"}, nil)

	svc := newAdminService(repo, new(mockRefreshTokenRepo))

	_, err := svc.CreateAdmin(context.Background(), "existing@example.com", "New Admin", "longenough1", "admin")
	assert.ErrorIs(t, err, utils.ErrConflict)
}

func TestAdminService_CreateAdmin_DefaultsRole(t *testing.T) {
	repo := new(mockAdminRepo)
	repo.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*models.Admin")).
		Run(func(args mock.Arguments) {
			a := args.Get(1).(*models.Admin)
			assert.Equal(t, "admin", a.Role)
		}).
		Return(nil)

	svc := newAdminService(repo, new(mockRefreshTokenRepo))

	admin, err := svc.CreateAdmin(context.Background(), "new@example.com", "New Admin", "longenough1", "")
	require.NoError(t, err)
	assert.Equal(t, "admin", admin.Role)
	assert.NotEqual(t, "longenough1", admin.PasswordHash)
}

func TestAdminService_UpdateAdmin_NotFound(t *testing.T) {
	repo := new(mockAdminRepo)
	repo.On("GetByID", mock.Anything, "missing").Return(nil, nil)

	svc := newAdminService(repo, new(mockRefreshTokenRepo))

	_, err := svc.UpdateAdmin(context.Background(), "missing", "New Name", "", "")
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestAdminService_UpdateAdmin_ShortPasswordRejected(t *testing.T) {
	repo := new(mockAdminRepo)
	repo.On("GetByID", mock.Anything, "admin-1").Return(&models.Admin{ID: "admin-1"}, nil)

	svc := newAdminService(repo, new(mockRefreshTokenRepo))

	_, err := svc.UpdateAdmin(context.Background(), "admin-1", "New Name", "short", "")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestAdminService_ChangePassword_WrongCurrentPasswordRejected(t *testing.T) {
	hash, err := utils.HashPassword("current-password")
	require.NoError(t, err)

	repo := new(mockAdminRepo)
	repo.On("GetByID", mock.Anything, "admin-1").
		Return(&models.Admin{ID: "admin-1", PasswordHash: hash}, nil)

	svc := newAdminService(repo, new(mockRefreshTokenRepo))

	err = svc.ChangePassword(context.Background(), "admin-1", "wrong-password", "newpassword1")
	assert.ErrorIs(t, err, utils.ErrUnauthorized)
}

func TestAdminService_ChangePassword_ShortNewPasswordRejected(t *testing.T) {
	svc := newAdminService(new(mockAdminRepo), new(mockRefreshTokenRepo))

	err := svc.ChangePassword(context.Background(), "admin-1", "current-password", "short")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestAdminService_ChangePassword_SamePasswordRejected(t *testing.T) {
	hash, err := utils.HashPassword("current-password")
	require.NoError(t, err)

	repo := new(mockAdminRepo)
	repo.On("GetByID", mock.Anything, "admin-1").
		Return(&models.Admin{ID: "admin-1", PasswordHash: hash}, nil)

	svc := newAdminService(repo, new(mockRefreshTokenRepo))

	err = svc.ChangePassword(context.Background(), "admin-1", "current-password", "current-password")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestAdminService_ChangePassword_SuccessRevokesAllSessions(t *testing.T) {
	hash, err := utils.HashPassword("current-password")
	require.NoError(t, err)

	repo := new(mockAdminRepo)
	repo.On("GetByID", mock.Anything, "admin-1").
		Return(&models.Admin{ID: "admin-1", PasswordHash: hash}, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Admin) bool {
		return a.ID == "admin-1" && utils.CheckPassword("newpassword1", a.PasswordHash)
	})).Return(nil)

	refreshRepo := new(mockRefreshTokenRepo)
	refreshRepo.On("RevokeAllForSubject", mock.Anything, "admin-1", models.SubjectTypeAdmin).Return(nil)

	svc := newAdminService(repo, refreshRepo)

	err = svc.ChangePassword(context.Background(), "admin-1", "current-password", "newpassword1")
	require.NoError(t, err)
	repo.AssertExpectations(t)
	refreshRepo.AssertCalled(t, "RevokeAllForSubject", mock.Anything, "admin-1", models.SubjectTypeAdmin)
}

func TestAdminService_DeleteAdmin_NotFound(t *testing.T) {
	repo := new(mockAdminRepo)
	repo.On("GetByID", mock.Anything, "missing").Return(nil, nil)

	svc := newAdminService(repo, new(mockRefreshTokenRepo))

	err := svc.DeleteAdmin(context.Background(), "missing")
	assert.ErrorIs(t, err, utils.ErrNotFound)
}
