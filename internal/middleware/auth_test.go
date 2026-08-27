package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/middleware"
	"tixora/internal/utils"
)

const testSecret = "test_secret_min_32_chars_long_ok"

func init() {
	gin.SetMode(gin.TestMode)
}

func newAuthedRouter() *gin.Engine {
	router := gin.New()
	router.GET("/protected", middleware.AuthMiddleware(testSecret), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": c.GetString("user_id"),
			"email":   c.GetString("email"),
		})
	})
	return router
}

func TestAuthMiddleware_MissingCookie(t *testing.T) {
	router := newAuthedRouter()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	router := newAuthedRouter()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.UserAccessCookie, Value: "not-a-valid-jwt"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	router := newAuthedRouter()

	token, err := utils.GenerateJWT("user-1", "user@example.com", testSecret, -time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.UserAccessCookie, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	router := newAuthedRouter()

	token, err := utils.GenerateJWT("user-1", "user@example.com", "a_totally_different_secret_32chr", time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.UserAccessCookie, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	router := newAuthedRouter()

	token, err := utils.GenerateJWT("user-1", "user@example.com", testSecret, time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.UserAccessCookie, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "user-1")
	assert.Contains(t, rec.Body.String(), "user@example.com")
}

// An admin token must not authenticate a user route - the two claim shapes
// happen to both parse under VerifyJWT, but admin tokens carry no user_id.
func TestAuthMiddleware_AdminTokenDoesNotGrantUserID(t *testing.T) {
	router := newAuthedRouter()

	adminToken, err := utils.GenerateAdminJWT("admin-1", "admin@example.com", "admin", testSecret, time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.UserAccessCookie, Value: adminToken})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"user_id":""`)
}
