package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"tixora/internal/middleware"
)

func newCORSRouter() *gin.Engine {
	router := gin.New()
	router.Use(middleware.CORSMiddleware([]string{"http://localhost:3000", "http://localhost:3001"}))
	router.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func TestCORSMiddleware_AllowedOriginEchoed(t *testing.T) {
	router := newCORSRouter()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_DisallowedOriginNotEchoed(t *testing.T) {
	router := newCORSRouter()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_PreflightShortCircuits(t *testing.T) {
	router := newCORSRouter()

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}
