package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"tixora/internal/dto"
	"tixora/internal/models"
	"tixora/internal/services"
)

// OrderHandler handles order placement and lifecycle endpoints.
type OrderHandler struct {
	orderService services.IOrderService
	imageBaseURL string
}

// NewOrderHandler builds an OrderHandler. imageBaseURL is the public/CDN
// base URL event images are served from, used to build absolute
// event_image values in responses.
func NewOrderHandler(orderService services.IOrderService, imageBaseURL string) *OrderHandler {
	return &OrderHandler{orderService: orderService, imageBaseURL: imageBaseURL}
}

// CreateOrder handles POST /api/orders
//
//	@Summary		Create order
//	@Description	Creates a new ticket order for the authenticated user.
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			request	body		dto.CreateOrderRequest	true	"Order details"
//	@Success		201		{object}	dto.SuccessResponse{data=dto.OrderResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	order, err := h.orderService.CreateOrder(c.Request.Context(), userID, services.CreateOrderInput{
		EventID:    req.EventID,
		Quantity:   req.Quantity,
		BuyerName:  req.BuyerName,
		BuyerEmail: req.BuyerEmail,
	})
	if err != nil {
		respondError(c, err, "Failed to create order")
		return
	}

	c.JSON(http.StatusCreated, dto.NewSuccessResponse("Order created", dto.NewOrderResponse(order, h.imageBaseURL)))
}

// GetUserOrders handles GET /api/orders (protected)
//
//	@Summary		List my orders
//	@Description	Returns a paginated list of orders belonging to the authenticated user.
//	@Tags			orders
//	@Produce		json
//	@Security		CookieAuth
//	@Param			page	query		int		false	"Page number"		default(1)
//	@Param			limit	query		int		false	"Items per page"	default(12)
//	@Param			status	query		string	false	"Filter by order status (pending, paid, expired, cancelled)"
//	@Success		200		{object}	dto.ListResponse{data=[]dto.OrderResponse}
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/orders [get]
func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 12)
	status := models.OrderStatus(c.Query("status"))

	orders, total, err := h.orderService.GetUserOrders(c.Request.Context(), userID, page, limit, status)
	if err != nil {
		respondError(c, err, "Failed to fetch orders")
		return
	}

	c.JSON(http.StatusOK, dto.NewListResponse(dto.NewOrderResponseList(orders, h.imageBaseURL), page, limit, int(total)))
}

// GetOrderByID handles GET /api/orders/:id (protected)
//
//	@Summary		Get order by ID
//	@Description	Returns a single order owned by the authenticated user.
//	@Tags			orders
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id	path		string	true	"Order ID"
//	@Success		200	{object}	dto.SuccessResponse{data=dto.OrderResponse}
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/orders/{id} [get]
func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	id := c.Param("id")

	order, err := h.orderService.GetOrderByID(c.Request.Context(), userID, id)
	if err != nil {
		respondError(c, err, "Failed to fetch order")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewOrderResponse(order, h.imageBaseURL)))
}

// GetOrderStatus handles GET /api/orders/:id/status (protected)
//
//	@Summary		Get order status
//	@Description	Returns a lightweight status view of an order, intended for polling.
//	@Tags			orders
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id	path		string	true	"Order ID"
//	@Success		200	{object}	dto.SuccessResponse{data=dto.OrderStatusResponse}
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/orders/{id}/status [get]
func (h *OrderHandler) GetOrderStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	id := c.Param("id")

	order, err := h.orderService.GetOrderByID(c.Request.Context(), userID, id)
	if err != nil {
		respondError(c, err, "Failed to fetch order status")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewOrderStatusResponse(order)))
}

// CancelOrder handles POST /api/orders/:id/cancel (protected)
//
//	@Summary		Cancel order
//	@Description	Cancels a pending order owned by the authenticated user and releases its reserved tickets.
//	@Tags			orders
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id	path		string	true	"Order ID"
//	@Success		200	{object}	dto.SuccessResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		409	{object}	dto.ErrorResponse
//	@Router			/orders/{id}/cancel [post]
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	id := c.Param("id")

	if err := h.orderService.CancelOrder(c.Request.Context(), userID, id); err != nil {
		respondError(c, err, "Failed to cancel order")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Order cancelled", nil))
}

// ContinuePayment handles POST /api/orders/:id/pay (protected)
//
//	@Summary		Continue payment
//	@Description	Returns the payment URL/Snap token for a pending order owned by the authenticated user, so they can resume an interrupted payment.
//	@Tags			orders
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id	path		string	true	"Order ID"
//	@Success		200	{object}	dto.SuccessResponse{data=dto.OrderResponse}
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		409	{object}	dto.ErrorResponse
//	@Router			/orders/{id}/pay [post]
func (h *OrderHandler) ContinuePayment(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	id := c.Param("id")

	order, err := h.orderService.ContinuePayment(c.Request.Context(), userID, id)
	if err != nil {
		respondError(c, err, "Failed to continue payment")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewOrderResponse(order, h.imageBaseURL)))
}

// DownloadTicket handles GET /api/orders/:id/ticket/download (protected)
//
//	@Summary		Download e-ticket
//	@Description	Returns a PDF e-ticket for a paid order owned by the authenticated user.
//	@Tags			orders
//	@Produce		application/pdf
//	@Security		CookieAuth
//	@Param			id	path	string	true	"Order ID"
//	@Success		200	{file}	file
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		409	{object}	dto.ErrorResponse
//	@Router			/orders/{id}/ticket/download [get]
func (h *OrderHandler) DownloadTicket(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	id := c.Param("id")

	order, pdf, err := h.orderService.GenerateTicketPDF(c.Request.Context(), userID, id)
	if err != nil {
		respondError(c, err, "Failed to generate ticket")
		return
	}

	filename := fmt.Sprintf("ticket-%s.pdf", order.OrderID)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", pdf)
}

// AdminListOrders handles GET /api/admin/orders (admin only)
//
//	@Summary		List all orders
//	@Description	Returns a paginated list of orders across all users. Requires admin authentication.
//	@Tags			admin
//	@Produce		json
//	@Security		CookieAuth
//	@Param			page	query		int		false	"Page number"		default(1)
//	@Param			limit	query		int		false	"Items per page"	default(12)
//	@Param			status	query		string	false	"Filter by order status (pending, paid, expired, cancelled)"
//	@Success		200		{object}	dto.ListResponse{data=[]dto.OrderResponse}
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/admin/orders [get]
func (h *OrderHandler) AdminListOrders(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 12)
	status := models.OrderStatus(c.Query("status"))

	orders, total, err := h.orderService.GetAllOrders(c.Request.Context(), page, limit, status)
	if err != nil {
		respondError(c, err, "Failed to fetch orders")
		return
	}

	c.JSON(http.StatusOK, dto.NewListResponse(dto.NewOrderResponseList(orders, h.imageBaseURL), page, limit, int(total)))
}

// AdminGetOrderByID handles GET /api/admin/orders/:id (admin only)
//
//	@Summary		Get order detail (admin)
//	@Description	Returns a single order by its order ID, regardless of owner. Requires admin authentication.
//	@Tags			admin
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id	path		string	true	"Order ID"
//	@Success		200	{object}	dto.SuccessResponse{data=dto.OrderResponse}
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/admin/orders/{id} [get]
func (h *OrderHandler) AdminGetOrderByID(c *gin.Context) {
	id := c.Param("id")

	order, err := h.orderService.GetOrderByOrderIDAdmin(c.Request.Context(), id)
	if err != nil {
		respondError(c, err, "Failed to fetch order")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewOrderResponse(order, h.imageBaseURL)))
}
