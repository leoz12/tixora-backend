package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"tixora/internal/middleware"
	"tixora/internal/utils"
)

func newCSRFProtectedRouter() *gin.Engine {
	router := gin.New()
	router.Use(middleware.CSRFMiddleware(utils.UserCSRFCookie))
	router.GET("/read", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/write", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.PUT("/write", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.DELETE("/write", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func TestCSRFMiddleware_SafeMethodsBypassCheck(t *testing.T) {
	router := newCSRFProtectedRouter()

	req := httptest.NewRequest(http.MethodGet, "/read", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCSRFMiddleware_PostWithoutTokenRejected(t *testing.T) {
	router := newCSRFProtectedRouter()

	req := httptest.NewRequest(http.MethodPost, "/write", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRFMiddleware_PostWithOnlyCookieRejected(t *testing.T) {
	router := newCSRFProtectedRouter()

	req := httptest.NewRequest(http.MethodPost, "/write", nil)
	req.AddCookie(&http.Cookie{Name: utils.UserCSRFCookie, Value: "csrf-token-value"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRFMiddleware_PostWithOnlyHeaderRejected(t *testing.T) {
	router := newCSRFProtectedRouter()

	req := httptest.NewRequest(http.MethodPost, "/write", nil)
	req.Header.Set("X-CSRF-Token", "csrf-token-value")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRFMiddleware_MismatchedTokenRejected(t *testing.T) {
	router := newCSRFProtectedRouter()

	req := httptest.NewRequest(http.MethodPost, "/write", nil)
	req.AddCookie(&http.Cookie{Name: utils.UserCSRFCookie, Value: "cookie-token"})
	req.Header.Set("X-CSRF-Token", "different-header-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRFMiddleware_MatchingTokenAccepted(t *testing.T) {
	router := newCSRFProtectedRouter()

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/write", nil)
		req.AddCookie(&http.Cookie{Name: utils.UserCSRFCookie, Value: "matching-token"})
		req.Header.Set("X-CSRF-Token", "matching-token")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "method %s should be accepted with a matching CSRF token", method)
	}
}
