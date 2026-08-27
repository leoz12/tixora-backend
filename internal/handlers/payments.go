package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tixora/internal/dto"
	"tixora/internal/services"
)

// PaymentHandler handles payment webhook and status endpoints.
type PaymentHandler struct {
	paymentService services.IPaymentService
}

func NewPaymentHandler(paymentService services.IPaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

// WebhookHandler handles POST /api/payments/webhook (Midtrans payment notification).
// No authentication is required here; the request is instead authenticated via
// the Midtrans signature verified inside the service layer.
//
//	@Summary		Midtrans payment webhook
//	@Description	Receives asynchronous payment notifications from Midtrans. Authenticated via the Midtrans signature key, not a bearer token.
//	@Tags			payments
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.MidtransWebhookRequest	true	"Midtrans notification payload"
//	@Success		200		{object}	dto.SuccessResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/payments/webhook [post]
func (h *PaymentHandler) WebhookHandler(c *gin.Context) {
	var req dto.MidtransWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid webhook payload", err))
		return
	}

	notification := services.MidtransNotification{
		OrderID:           req.OrderID,
		StatusCode:        req.StatusCode,
		GrossAmount:       req.GrossAmount,
		SignatureKey:      req.SignatureKey,
		TransactionStatus: req.TransactionStatus,
		TransactionID:     req.TransactionID,
		PaymentType:       req.PaymentType,
		FraudStatus:       req.FraudStatus,
	}

	if err := h.paymentService.ProcessWebhook(c.Request.Context(), notification); err != nil {
		respondError(c, err, "Failed to process payment notification")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Payment notification processed", nil))
}

// GetPaymentStatus handles GET /api/payments/status/:transactionId (protected)
//
//	@Summary		Get payment status
//	@Description	Returns the status of a payment transaction owned by the authenticated user.
//	@Tags			payments
//	@Produce		json
//	@Security		CookieAuth
//	@Param			transactionId	path		string	true	"Midtrans transaction ID"
//	@Success		200				{object}	dto.SuccessResponse{data=dto.PaymentStatusResponse}
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Router			/payments/status/{transactionId} [get]
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	transactionID := c.Param("transactionId")

	payment, err := h.paymentService.GetPaymentStatus(c.Request.Context(), userID, transactionID)
	if err != nil {
		respondError(c, err, "Failed to fetch payment status")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewPaymentStatusResponse(payment)))
}
