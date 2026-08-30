package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/dto"
)

func TestE2E_Orders_CreateRequiresAuth(t *testing.T) {
	app := newTestApp(t)

	req := map[string]interface{}{
		"event_id": "evt-1", "quantity": 2, "buyer_name": "Jane Doe", "buyer_email": "jane@example.com",
	}
	rec := app.do(t, http.MethodPost, "/api/orders", req, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestE2E_Orders_FullLifecycle(t *testing.T) {
	app := newTestApp(t)
	category := app.seedCategory(t, "cat-1", "Music", "music")
	event := app.seedEvent(t, "evt-1", category.ID, 10)
	app.seedUser(t, "user-1", "jane@example.com")
	cookies := app.userCookies(t, "user-1", "jane@example.com")

	// Create.
	createReq := map[string]interface{}{
		"event_id": event.ID, "quantity": 2, "buyer_name": "Jane Doe", "buyer_email": "jane@example.com",
	}
	var created struct {
		Success bool              `json:"success"`
		Data    dto.OrderResponse `json:"data"`
	}
	rec := app.do(t, http.MethodPost, "/api/orders", createReq, &cookies, &created)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "pending", created.Data.Status)
	assert.EqualValues(t, 205000, created.Data.TotalPrice)
	assert.NotEmpty(t, created.Data.SnapToken, "stub payment service must still populate a snap token")
	orderID := created.Data.OrderID
	require.NotEmpty(t, orderID)

	// Ticket count must have been reserved on the event.
	var eventAfter struct {
		Success bool              `json:"success"`
		Data    dto.EventResponse `json:"data"`
	}
	rec = app.do(t, http.MethodGet, "/api/events/"+event.ID, nil, nil, &eventAfter)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 8, eventAfter.Data.AvailableTickets)

	// Get by ID.
	var fetched struct {
		Success bool              `json:"success"`
		Data    dto.OrderResponse `json:"data"`
	}
	rec = app.do(t, http.MethodGet, "/api/orders/"+orderID, nil, &cookies, &fetched)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, orderID, fetched.Data.OrderID)

	// List my orders.
	var list dto.ListResponse
	rec = app.do(t, http.MethodGet, "/api/orders", nil, &cookies, &list)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.EqualValues(t, 1, list.Pagination.TotalItems)

	// Cancel - must release the reserved tickets.
	rec = app.do(t, http.MethodPost, "/api/orders/"+orderID+"/cancel", nil, &cookies, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = app.do(t, http.MethodGet, "/api/events/"+event.ID, nil, nil, &eventAfter)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 10, eventAfter.Data.AvailableTickets, "cancelling must release the 2 reserved tickets")

	// Cancelling again must fail (no longer pending).
	rec = app.do(t, http.MethodPost, "/api/orders/"+orderID+"/cancel", nil, &cookies, nil)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestE2E_Orders_CreateRejectsInsufficientTickets(t *testing.T) {
	app := newTestApp(t)
	category := app.seedCategory(t, "cat-1", "Music", "music")
	event := app.seedEvent(t, "evt-1", category.ID, 1)
	app.seedUser(t, "user-1", "jane@example.com")
	cookies := app.userCookies(t, "user-1", "jane@example.com")

	req := map[string]interface{}{
		"event_id": event.ID, "quantity": 5, "buyer_name": "Jane Doe", "buyer_email": "jane@example.com",
	}
	rec := app.do(t, http.MethodPost, "/api/orders", req, &cookies, nil)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestE2E_Orders_GetOtherUsersOrderForbidden(t *testing.T) {
	app := newTestApp(t)
	category := app.seedCategory(t, "cat-1", "Music", "music")
	event := app.seedEvent(t, "evt-1", category.ID, 10)
	app.seedUser(t, "user-1", "owner@example.com")
	app.seedUser(t, "user-2", "intruder@example.com")

	ownerCookies := app.userCookies(t, "user-1", "owner@example.com")
	createReq := map[string]interface{}{
		"event_id": event.ID, "quantity": 1, "buyer_name": "Owner", "buyer_email": "owner@example.com",
	}
	var created struct {
		Success bool              `json:"success"`
		Data    dto.OrderResponse `json:"data"`
	}
	rec := app.do(t, http.MethodPost, "/api/orders", createReq, &ownerCookies, &created)
	require.Equal(t, http.StatusCreated, rec.Code)

	intruderCookies := app.userCookies(t, "user-2", "intruder@example.com")
	rec = app.do(t, http.MethodGet, "/api/orders/"+created.Data.OrderID, nil, &intruderCookies, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestE2E_Orders_CreateRejectsCrossSiteOrigin(t *testing.T) {
	app := newTestApp(t)
	category := app.seedCategory(t, "cat-1", "Music", "music")
	event := app.seedEvent(t, "evt-1", category.ID, 10)
	app.seedUser(t, "user-1", "jane@example.com")

	cookies := app.userCookies(t, "user-1", "jane@example.com")
	cookies.origin = "https://evil.example" // cross-site request, not in the CORS allowlist

	req := map[string]interface{}{
		"event_id": event.ID, "quantity": 1, "buyer_name": "Jane Doe", "buyer_email": "jane@example.com",
	}
	rec := app.do(t, http.MethodPost, "/api/orders", req, &cookies, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
