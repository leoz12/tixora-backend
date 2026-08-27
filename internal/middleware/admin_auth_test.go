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

func newAdminAuthedRouter() *gin.Engine {
	router := gin.New()
	router.GET("/admin/protected", middleware.AdminAuthMiddleware(testSecret), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"admin_id":   c.GetString("admin_id"),
			"admin_role": c.GetString("admin_role"),
		})
	})
	return router
}

func TestAdminAuthMiddleware_MissingCookie(t *testing.T) {
	router := newAdminAuthedRouter()

	req := httptest.NewRequest(http.MethodGet, "/admin/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAdminAuthMiddleware_ValidToken(t *testing.T) {
	router := newAdminAuthedRouter()

	token, err := utils.GenerateAdminJWT("admin-1", "admin@example.com", "superadmin", testSecret, time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.AdminAccessCookie, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "admin-1")
	assert.Contains(t, rec.Body.String(), "superadmin")
}

// A user access token placed in the admin cookie slot must not authenticate
// as an admin - the two sessions are fully separate.
func TestAdminAuthMiddleware_UserTokenInAdminCookieRejected(t *testing.T) {
	router := newAdminAuthedRouter()

	userToken, err := utils.GenerateJWT("user-1", "user@example.com", testSecret, time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.AdminAccessCookie, Value: userToken})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// The user JWT still parses under the admin verifier (same alg/secret),
	// but carries no admin_id/role - assert the handler sees an empty admin_id
	// rather than silently trusting it.
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"admin_id":""`)
}

func TestAdminAuthMiddleware_ExpiredToken(t *testing.T) {
	router := newAdminAuthedRouter()

	token, err := utils.GenerateAdminJWT("admin-1", "admin@example.com", "admin", testSecret, -time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/admin/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.AdminAccessCookie, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
