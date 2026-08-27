package e2e

import (
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/dto"
)

func webhookSignature(orderID, statusCode, grossAmount string) string {
	sum := sha512.Sum512([]byte(orderID + statusCode + grossAmount + testMidtransServerKey))
	return hex.EncodeToString(sum[:])
}

func TestE2E_Payments_WebhookMarksOrderPaid(t *testing.T) {
	app := newTestApp(t)
	category := app.seedCategory(t, "cat-1", "Music", "music")
	event := app.seedEvent(t, "evt-1", category.ID, 10)
	app.seedUser(t, "user-1", "jane@example.com")
	cookies := app.userCookies(t, "user-1", "jane@example.com")

	createReq := map[string]interface{}{
		"event_id": event.ID, "quantity": 1, "buyer_name": "Jane Doe", "buyer_email": "jane@example.com",
	}
	var created struct {
		Success bool              `json:"success"`
		Data    dto.OrderResponse `json:"data"`
	}
	rec := app.do(t, http.MethodPost, "/api/orders", createReq, &cookies, &created)
	require.Equal(t, http.StatusCreated, rec.Code)
	orderID := created.Data.OrderID

	webhook := map[string]interface{}{
		"order_id": orderID, "status_code": "200", "gross_amount": "205000.00",
		"transaction_status": "settlement", "transaction_id": "txn-e2e-1", "payment_type": "bank_transfer",
	}
	webhook["signature_key"] = webhookSignature(orderID, "200", "205000.00")

	rec = app.do(t, http.MethodPost, "/api/payments/webhook", webhook, nil, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var fetched struct {
		Success bool              `json:"success"`
		Data    dto.OrderResponse `json:"data"`
	}
	rec = app.do(t, http.MethodGet, "/api/orders/"+orderID, nil, &cookies, &fetched)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "paid", fetched.Data.Status)
	require.NotNil(t, fetched.Data.TicketReference)
	assert.NotEmpty(t, *fetched.Data.TicketReference)
}

func TestE2E_Payments_WebhookRejectsInvalidSignature(t *testing.T) {
	app := newTestApp(t)

	webhook := map[string]interface{}{
		"order_id": "ORD-FAKE", "status_code": "200", "gross_amount": "205000.00",
		"transaction_status": "settlement", "transaction_id": "txn-fake", "signature_key": "not-a-real-signature",
	}

	rec := app.do(t, http.MethodPost, "/api/payments/webhook", webhook, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestE2E_Payments_GetStatusRequiresAuth(t *testing.T) {
	app := newTestApp(t)

	rec := app.do(t, http.MethodGet, "/api/payments/status/txn-1", nil, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
