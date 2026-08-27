package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tixora/internal/dto"
	"tixora/internal/services"
)

// UserHandler handles user profile endpoints.
type UserHandler struct {
	userService  services.IUserService
	orderService services.IOrderService
}

func NewUserHandler(userService services.IUserService, orderService services.IOrderService) *UserHandler {
	return &UserHandler{userService: userService, orderService: orderService}
}

// GetProfile handles GET /api/user/profile (protected)
//
//	@Summary		Get profile
//	@Description	Returns the profile of the authenticated user.
//	@Tags			user
//	@Produce		json
//	@Security		CookieAuth
//	@Success		200	{object}	dto.SuccessResponse{data=dto.UserResponse}
//	@Failure		401	{object}	dto.ErrorResponse
//	@Router			/user/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err, "Failed to fetch profile")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewUserResponse(user)))
}

// UpdateProfile handles PUT /api/user/profile (protected)
//
//	@Summary		Update profile
//	@Description	Updates the profile of the authenticated user.
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			request	body		dto.UpdateProfileRequest	true	"Profile fields"
//	@Success		200		{object}	dto.SuccessResponse{data=dto.UserResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/user/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	user, err := h.userService.UpdateProfile(c.Request.Context(), userID, req.Name, req.AvatarURL)
	if err != nil {
		respondError(c, err, "Failed to update profile")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Profile updated", dto.NewUserResponse(user)))
}

// GetStats handles GET /api/user/stats (protected)
//
//	@Summary		Get order stats
//	@Description	Returns the authenticated user's paid order count and total amount spent.
//	@Tags			user
//	@Produce		json
//	@Security		CookieAuth
//	@Success		200	{object}	dto.SuccessResponse{data=dto.UserStatsResponse}
//	@Failure		401	{object}	dto.ErrorResponse
//	@Router			/user/stats [get]
func (h *UserHandler) GetStats(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	totalOrders, totalSpent, err := h.orderService.GetUserOrderStats(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err, "Failed to fetch order stats")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.UserStatsResponse{
		TotalOrders: totalOrders,
		TotalSpent:  totalSpent,
	}))
}

// AdminListUsers handles GET /api/admin/users (admin only)
//
//	@Summary		List customers
//	@Description	Returns a paginated list of registered users. Requires admin authentication.
//	@Tags			admin
//	@Produce		json
//	@Security		CookieAuth
//	@Param			page	query		int		false	"Page number"		default(1)
//	@Param			limit	query		int		false	"Items per page"	default(20)
//	@Param			search	query		string	false	"Filter by name or email"
//	@Success		200		{object}	dto.ListResponse{data=[]dto.UserResponse}
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/admin/users [get]
func (h *UserHandler) AdminListUsers(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 20)
	search := c.Query("search")

	users, total, err := h.userService.ListUsers(c.Request.Context(), page, limit, search)
	if err != nil {
		respondError(c, err, "Failed to fetch users")
		return
	}

	c.JSON(http.StatusOK, dto.NewListResponse(dto.NewUserResponseList(users), page, limit, int(total)))
}

// AdminGetUserByID handles GET /api/admin/users/:id (admin only)
//
//	@Summary		Get customer detail
//	@Description	Returns a single user by ID. Requires admin authentication.
//	@Tags			admin
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	dto.SuccessResponse{data=dto.UserResponse}
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/admin/users/{id} [get]
func (h *UserHandler) AdminGetUserByID(c *gin.Context) {
	id := c.Param("id")

	user, err := h.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err, "Failed to fetch user")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewUserResponse(user)))
}
