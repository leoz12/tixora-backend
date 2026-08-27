package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/dto"
)

func TestE2E_Categories_PublicListExcludesInactive(t *testing.T) {
	app := newTestApp(t)
	app.seedCategory(t, "cat-1", "Music", "music")

	var list struct {
		Success bool                   `json:"success"`
		Data    []dto.CategoryResponse `json:"data"`
	}
	rec := app.do(t, http.MethodGet, "/api/categories", nil, nil, &list)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, list.Data, 1)
	assert.Equal(t, "Music", list.Data[0].Name)
}

func TestE2E_Categories_AdminCreateFlow(t *testing.T) {
	app := newTestApp(t)
	app.seedAdmin(t, "admin-1", "admin@example.com", "password123", "admin")
	cookies := app.adminCookies(t, "admin-1", "admin@example.com", "admin")

	var created struct {
		Success bool                 `json:"success"`
		Data    dto.CategoryResponse `json:"data"`
	}
	rec := app.do(t, http.MethodPost, "/api/categories", map[string]string{"name": "Sports"}, &cookies, &created)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "Sports", created.Data.Name)
	assert.Equal(t, "sports", created.Data.Slug)
}

func TestE2E_Categories_CreateDuplicateNameConflict(t *testing.T) {
	app := newTestApp(t)
	app.seedCategory(t, "cat-1", "Sports", "sports")
	app.seedAdmin(t, "admin-1", "admin@example.com", "password123", "admin")
	cookies := app.adminCookies(t, "admin-1", "admin@example.com", "admin")

	rec := app.do(t, http.MethodPost, "/api/categories", map[string]string{"name": "Sports"}, &cookies, nil)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestE2E_Categories_DeleteInUseConflict(t *testing.T) {
	app := newTestApp(t)
	category := app.seedCategory(t, "cat-1", "Music", "music")
	app.seedEvent(t, "evt-1", category.ID, 10)
	app.seedAdmin(t, "admin-1", "admin@example.com", "password123", "admin")
	cookies := app.adminCookies(t, "admin-1", "admin@example.com", "admin")

	rec := app.do(t, http.MethodDelete, "/api/categories/"+category.ID, nil, &cookies, nil)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestE2E_Categories_DeleteRequiresAdmin(t *testing.T) {
	app := newTestApp(t)
	category := app.seedCategory(t, "cat-1", "Music", "music")

	rec := app.do(t, http.MethodDelete, "/api/categories/"+category.ID, nil, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
