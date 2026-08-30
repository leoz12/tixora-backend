package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"tixora/internal/middleware"
)

var csrfAllowedOrigins = []string{"https://app.example.com", "https://admin.example.com"}

func newCSRFProtectedRouter() *gin.Engine {
	router := gin.New()
	router.Use(middleware.CSRFMiddleware(csrfAllowedOrigins))
	router.GET("/read", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/write", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.PUT("/write", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.DELETE("/write", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func TestCSRFMiddleware_SafeMethodsBypassCheck(t *testing.T) {
	router := newCSRFProtectedRouter()

	req := httptest.NewRequest(http.MethodGet, "/read", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCSRFMiddleware_AllowedOriginAccepted(t *testing.T) {
	router := newCSRFProtectedRouter()

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/write", nil)
		req.Header.Set("Origin", "https://admin.example.com")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "method %s from an allowed origin should pass", method)
	}
}

func TestCSRFMiddleware_DisallowedOriginRejected(t *testing.T) {
	router := newCSRFProtectedRouter()

	req := httptest.NewRequest(http.MethodPost, "/write", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRFMiddleware_FallsBackToRefererWhenNoOrigin(t *testing.T) {
	router := newCSRFProtectedRouter()

	allowed := httptest.NewRequest(http.MethodPost, "/write", nil)
	allowed.Header.Set("Referer", "https://app.example.com/dashboard")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, allowed)
	assert.Equal(t, http.StatusOK, rec.Code)

	denied := httptest.NewRequest(http.MethodPost, "/write", nil)
	denied.Header.Set("Referer", "https://evil.example/attack")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, denied)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRFMiddleware_NoOriginOrRefererAllowed(t *testing.T) {
	router := newCSRFProtectedRouter()

	// Non-browser clients (curl, server-to-server) send neither header and
	// can't be tricked into riding a victim's cookie - not a CSRF vector.
	req := httptest.NewRequest(http.MethodPost, "/write", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
