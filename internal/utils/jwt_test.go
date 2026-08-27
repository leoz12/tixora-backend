package utils_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/utils"
)

const testJWTSecret = "test_secret_min_32_chars_long_ok"

func TestGenerateAndVerifyJWT(t *testing.T) {
	token, err := utils.GenerateJWT("user-1", "user@example.com", testJWTSecret, time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := utils.VerifyJWT(token, testJWTSecret)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "user@example.com", claims.Email)
}

func TestVerifyJWT_WrongSecret(t *testing.T) {
	token, err := utils.GenerateJWT("user-1", "user@example.com", testJWTSecret, time.Minute)
	require.NoError(t, err)

	_, err = utils.VerifyJWT(token, "a_completely_different_secret_32c")
	assert.Error(t, err)
}

func TestVerifyJWT_Expired(t *testing.T) {
	token, err := utils.GenerateJWT("user-1", "user@example.com", testJWTSecret, -time.Minute)
	require.NoError(t, err)

	_, err = utils.VerifyJWT(token, testJWTSecret)
	assert.Error(t, err)
}

func TestVerifyJWT_Malformed(t *testing.T) {
	_, err := utils.VerifyJWT("not-a-jwt", testJWTSecret)
	assert.Error(t, err)
}

func TestGenerateAndVerifyAdminJWT(t *testing.T) {
	token, err := utils.GenerateAdminJWT("admin-1", "admin@example.com", "superadmin", testJWTSecret, time.Minute)
	require.NoError(t, err)

	claims, err := utils.VerifyAdminJWT(token, testJWTSecret)
	require.NoError(t, err)
	assert.Equal(t, "admin-1", claims.AdminID)
	assert.Equal(t, "admin@example.com", claims.Email)
	assert.Equal(t, "superadmin", claims.Role)
}

func TestVerifyAdminJWT_Expired(t *testing.T) {
	token, err := utils.GenerateAdminJWT("admin-1", "admin@example.com", "admin", testJWTSecret, -time.Minute)
	require.NoError(t, err)

	_, err = utils.VerifyAdminJWT(token, testJWTSecret)
	assert.Error(t, err)
}

// A user token must not verify as an admin token and vice versa - they carry
// different claim shapes and the middleware for one must reject the other's cookie.
func TestUserAndAdminJWT_ClaimsAreDistinct(t *testing.T) {
	userToken, err := utils.GenerateJWT("user-1", "user@example.com", testJWTSecret, time.Minute)
	require.NoError(t, err)

	adminClaims, err := utils.VerifyAdminJWT(userToken, testJWTSecret)
	require.NoError(t, err) // token still parses (same secret/alg)...
	assert.Empty(t, adminClaims.AdminID, "a user token must not carry an admin id")
	assert.Empty(t, adminClaims.Role, "a user token must not carry a role")
}

func TestGenerateRefreshToken_UniqueAndHex(t *testing.T) {
	tok1, err := utils.GenerateRefreshToken()
	require.NoError(t, err)
	tok2, err := utils.GenerateRefreshToken()
	require.NoError(t, err)

	assert.NotEqual(t, tok1, tok2)
	assert.Len(t, tok1, 64) // 32 bytes hex-encoded
}

func TestHashToken_Deterministic(t *testing.T) {
	h1 := utils.HashToken("raw-token-value")
	h2 := utils.HashToken("raw-token-value")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64) // sha256 hex digest length

	h3 := utils.HashToken("different-token-value")
	assert.NotEqual(t, h1, h3)
}
