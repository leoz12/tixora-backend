package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/dto"
)

func TestE2E_Events_PublicListAndGet(t *testing.T) {
	app := newTestApp(t)
	category := app.seedCategory(t, "cat-1", "Music", "music")
	app.seedEvent(t, "evt-1", category.ID, 50)

	var list dto.ListResponse
	rec := app.do(t, http.MethodGet, "/api/events", nil, nil, &list)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.EqualValues(t, 1, list.Pagination.TotalItems)

	var single struct {
		Success bool              `json:"success"`
		Data    dto.EventResponse `json:"data"`
	}
	rec = app.do(t, http.MethodGet, "/api/events/evt-1", nil, nil, &single)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Test Event evt-1", single.Data.Title)
	assert.Equal(t, category.ID, single.Data.CategoryID)
}

func TestE2E_Events_GetByID_NotFound(t *testing.T) {
	app := newTestApp(t)

	rec := app.do(t, http.MethodGet, "/api/events/does-not-exist", nil, nil, nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestE2E_Events_CreateRequiresAdminAuth(t *testing.T) {
	app := newTestApp(t)

	req := map[string]interface{}{
		"title": "Unauthorized Event", "event_date": time.Now().Add(time.Hour),
		"location": "Jakarta", "price": 1000, "total_tickets": 10, "category_id": "cat-1",
	}

	rec := app.do(t, http.MethodPost, "/api/events", req, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestE2E_Events_CreateRequiresCSRFToken(t *testing.T) {
	app := newTestApp(t)
	app.seedAdmin(t, "admin-1", "admin@example.com", "password123", "admin")
	cookies := app.adminCookies(t, "admin-1", "admin@example.com", "admin")

	req := map[string]interface{}{
		"title": "Missing CSRF Event", "event_date": time.Now().Add(time.Hour),
		"location": "Jakarta", "price": 1000, "total_tickets": 10, "category_id": "cat-1",
	}

	// Strip the CSRF header - only the cookie is sent - which the
	// double-submit check must reject.
	noCSRF := cookies
	noCSRF.csrfToken = ""
	rec := app.do(t, http.MethodPost, "/api/events", req, &noCSRF, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestE2E_Events_AdminCreateUpdateDeleteFlow(t *testing.T) {
	app := newTestApp(t)
	category := app.seedCategory(t, "cat-1", "Music", "music")
	app.seedAdmin(t, "admin-1", "admin@example.com", "password123", "admin")
	cookies := app.adminCookies(t, "admin-1", "admin@example.com", "admin")

	createReq := map[string]interface{}{
		"title": "New Concert", "event_date": time.Now().Add(48 * time.Hour),
		"location": "Bandung", "price": 150000, "total_tickets": 200, "category_id": category.ID,
	}

	var created struct {
		Success bool              `json:"success"`
		Data    dto.EventResponse `json:"data"`
	}
	rec := app.do(t, http.MethodPost, "/api/events", createReq, &cookies, &created)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "New Concert", created.Data.Title)
	assert.Equal(t, 200, created.Data.AvailableTickets)
	eventID := created.Data.ID
	require.NotEmpty(t, eventID)

	updateReq := map[string]interface{}{
		"title": "Updated Concert", "event_date": time.Now().Add(72 * time.Hour),
		"location": "Bandung", "price": 175000, "total_tickets": 200, "category_id": category.ID,
	}
	var updated struct {
		Success bool              `json:"success"`
		Data    dto.EventResponse `json:"data"`
	}
	rec = app.do(t, http.MethodPut, "/api/events/"+eventID, updateReq, &cookies, &updated)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Updated Concert", updated.Data.Title)
	assert.EqualValues(t, 175000, updated.Data.Price)

	rec = app.do(t, http.MethodDelete, "/api/events/"+eventID, nil, &cookies, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = app.do(t, http.MethodGet, "/api/events/"+eventID, nil, nil, nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestE2E_Events_CreateRejectsUnknownCategory(t *testing.T) {
	app := newTestApp(t)
	app.seedAdmin(t, "admin-1", "admin@example.com", "password123", "admin")
	cookies := app.adminCookies(t, "admin-1", "admin@example.com", "admin")

	req := map[string]interface{}{
		"title": "Orphan Event", "event_date": time.Now().Add(time.Hour),
		"location": "Jakarta", "price": 1000, "total_tickets": 10, "category_id": "does-not-exist",
	}

	rec := app.do(t, http.MethodPost, "/api/events", req, &cookies, nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
