package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"tixora/internal/middleware"
)

func TestErrorMiddleware_RecoversFromPanic(t *testing.T) {
	router := gin.New()
	router.Use(middleware.ErrorMiddleware())
	router.GET("/boom", func(c *gin.Context) {
		panic("something went very wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		router.ServeHTTP(rec, req)
	})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestErrorMiddleware_TranslatesAttachedError(t *testing.T) {
	router := gin.New()
	router.Use(middleware.ErrorMiddleware())
	router.GET("/fail", func(c *gin.Context) {
		c.Status(http.StatusBadRequest)
		c.Error(errors.New("bad input"))
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "bad input")
}

func TestErrorMiddleware_NoErrorPassesThrough(t *testing.T) {
	router := gin.New()
	router.Use(middleware.ErrorMiddleware())
	router.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
