package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tixora/internal/config"
	"tixora/internal/dto"
	"tixora/internal/services"
	"tixora/internal/utils"
)

// AdminHandler handles admin authentication and management endpoints.
type AdminHandler struct {
	adminService services.IAdminService
	cfg          *config.Config
}

func NewAdminHandler(adminService services.IAdminService, cfg *config.Config) *AdminHandler {
	return &AdminHandler{adminService: adminService, cfg: cfg}
}

// Login handles POST /api/admin/auth/login
//
//	@Summary		Admin login
//	@Description	Authenticates an admin with email and password. Sets httpOnly access/refresh cookies.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.AdminLoginRequest	true	"Admin credentials"
//	@Success		200		{object}	dto.SuccessResponse{data=dto.AdminAuthResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/admin/auth/login [post]
func (h *AdminHandler) Login(c *gin.Context) {
	var req dto.AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	accessToken, refreshToken, admin, err := h.adminService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		respondError(c, err, "Failed to login")
		return
	}

	utils.SetAuthCookies(
		c, h.cfg,
		utils.AdminAccessCookie, utils.AdminRefreshCookie,
		accessToken, refreshToken,
		h.cfg.JWTAccessExpiry, h.cfg.JWTRefreshExpiry,
	)

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Login successful", dto.NewAdminAuthResponse(admin)))
}

// RefreshToken handles POST /api/admin/auth/refresh
//
//	@Summary		Refresh admin session
//	@Description	Rotates the admin refresh token cookie and issues a new access token.
//	@Tags			admin
//	@Produce		json
//	@Success		200	{object}	dto.SuccessResponse{data=dto.AdminAuthResponse}
//	@Failure		401	{object}	dto.ErrorResponse
//	@Router			/admin/auth/refresh [post]
func (h *AdminHandler) RefreshToken(c *gin.Context) {
	rawRefreshToken, _ := c.Cookie(utils.AdminRefreshCookie)

	accessToken, refreshToken, admin, err := h.adminService.RefreshToken(c.Request.Context(), rawRefreshToken)
	if err != nil {
		utils.ClearAuthCookies(c, h.cfg, utils.AdminAccessCookie, utils.AdminRefreshCookie)
		respondError(c, err, "Failed to refresh session")
		return
	}

	utils.SetAuthCookies(
		c, h.cfg,
		utils.AdminAccessCookie, utils.AdminRefreshCookie,
		accessToken, refreshToken,
		h.cfg.JWTAccessExpiry, h.cfg.JWTRefreshExpiry,
	)

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewAdminAuthResponse(admin)))
}

// Logout handles POST /api/admin/auth/logout
//
//	@Summary		Admin logout
//	@Description	Revokes the current admin refresh token and clears session cookies.
//	@Tags			admin
//	@Produce		json
//	@Success		200	{object}	dto.SuccessResponse
//	@Router			/admin/auth/logout [post]
func (h *AdminHandler) Logout(c *gin.Context) {
	rawRefreshToken, _ := c.Cookie(utils.AdminRefreshCookie)

	if err := h.adminService.Logout(c.Request.Context(), rawRefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("Failed to logout", err))
		return
	}

	utils.ClearAuthCookies(c, h.cfg, utils.AdminAccessCookie, utils.AdminRefreshCookie)
	c.JSON(http.StatusOK, dto.NewSuccessResponse("Logout successful", nil))
}

// GetCurrentAdmin handles GET /api/admin/auth/me (protected)
//
//	@Summary		Get current admin
//	@Description	Returns the profile of the currently authenticated admin.
//	@Tags			admin
//	@Produce		json
//	@Security		CookieAuth
//	@Success		200	{object}	dto.SuccessResponse{data=dto.AdminResponse}
//	@Failure		401	{object}	dto.ErrorResponse
//	@Router			/admin/auth/me [get]
func (h *AdminHandler) GetCurrentAdmin(c *gin.Context) {
	adminID := c.GetString("admin_id")
	if adminID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated admin", nil))
		return
	}

	admin, err := h.adminService.GetAdminByID(c.Request.Context(), adminID)
	if err != nil {
		respondError(c, err, "Failed to fetch current admin")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewAdminResponse(admin)))
}

// ChangePassword handles POST /api/admin/auth/change-password (protected)
//
//	@Summary		Change own password
//	@Description	Changes the currently authenticated admin's password. Requires the current password. On success every admin session (including this one) is revoked and the session cookies are cleared, so the client must log in again.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			request	body		dto.ChangeAdminPasswordRequest	true	"Current and new password"
//	@Success		200		{object}	dto.SuccessResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/admin/auth/change-password [post]
func (h *AdminHandler) ChangePassword(c *gin.Context) {
	adminID := c.GetString("admin_id")
	if adminID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated admin", nil))
		return
	}

	var req dto.ChangeAdminPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	if err := h.adminService.ChangePassword(c.Request.Context(), adminID, req.CurrentPassword, req.NewPassword); err != nil {
		respondError(c, err, "Failed to change password")
		return
	}

	// The change revoked every refresh token for this admin. Clear the
	// session cookies so the client falls back through the login flow.
	utils.ClearAuthCookies(c, h.cfg, utils.AdminAccessCookie, utils.AdminRefreshCookie)

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Password changed. Please log in again.", nil))
}

// ListAdmins handles GET /api/admin/admins (protected)
//
//	@Summary		List admins
//	@Description	Returns a paginated list of admins.
//	@Tags			admin
//	@Produce		json
//	@Security		CookieAuth
//	@Param			page	query		int	false	"Page number"		default(1)
//	@Param			limit	query		int	false	"Items per page"	default(20)
//	@Success		200		{object}	dto.ListResponse{data=[]dto.AdminResponse}
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/admin/admins [get]
func (h *AdminHandler) ListAdmins(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 20)

	admins, total, err := h.adminService.ListAdmins(c.Request.Context(), page, limit)
	if err != nil {
		respondError(c, err, "Failed to fetch admins")
		return
	}

	c.JSON(http.StatusOK, dto.NewListResponse(dto.NewAdminResponseList(admins), page, limit, int(total)))
}

// CreateAdmin handles POST /api/admin/admins (protected)
//
//	@Summary		Create admin
//	@Description	Creates a new admin account.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			request	body		dto.CreateAdminRequest	true	"New admin details"
//	@Success		201		{object}	dto.SuccessResponse{data=dto.AdminResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/admin/admins [post]
func (h *AdminHandler) CreateAdmin(c *gin.Context) {
	var req dto.CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	admin, err := h.adminService.CreateAdmin(c.Request.Context(), req.Email, req.Name, req.Password, req.Role)
	if err != nil {
		respondError(c, err, "Failed to create admin")
		return
	}

	c.JSON(http.StatusCreated, dto.NewSuccessResponse("Admin created", dto.NewAdminResponse(admin)))
}

// UpdateAdmin handles PUT /api/admin/admins/:id (protected)
//
//	@Summary		Update admin
//	@Description	Updates an existing admin account.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id		path		string					true	"Admin ID"
//	@Param			request	body		dto.UpdateAdminRequest	true	"Admin fields to update"
//	@Success		200		{object}	dto.SuccessResponse{data=dto.AdminResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Router			/admin/admins/{id} [put]
func (h *AdminHandler) UpdateAdmin(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	admin, err := h.adminService.UpdateAdmin(c.Request.Context(), id, req.Name, req.Password, req.Role)
	if err != nil {
		respondError(c, err, "Failed to update admin")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Admin updated", dto.NewAdminResponse(admin)))
}

// DeleteAdmin handles DELETE /api/admin/admins/:id (protected)
//
//	@Summary		Delete admin
//	@Description	Deletes an admin account.
//	@Tags			admin
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id	path		string	true	"Admin ID"
//	@Success		200	{object}	dto.SuccessResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/admin/admins/{id} [delete]
func (h *AdminHandler) DeleteAdmin(c *gin.Context) {
	id := c.Param("id")

	if err := h.adminService.DeleteAdmin(c.Request.Context(), id); err != nil {
		respondError(c, err, "Failed to delete admin")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Admin deleted", nil))
}
