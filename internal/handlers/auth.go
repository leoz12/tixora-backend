package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"tixora/internal/config"
	"tixora/internal/dto"
	"tixora/internal/services"
	"tixora/internal/utils"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService services.IAuthService
	userService services.IUserService
	cfg         *config.Config
}

func NewAuthHandler(authService services.IAuthService, userService services.IUserService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
		cfg:         cfg,
	}
}

// GoogleCallback handles POST /api/auth/oauth/google/callback
//
//	@Summary		Login with Google OAuth
//	@Description	Exchanges a Google OAuth authorization code for a Tixora session (creates the user if needed). Sets httpOnly access/refresh cookies.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.GoogleCallbackRequest	true	"Google OAuth code"
//	@Success		200		{object}	dto.SuccessResponse{data=dto.AuthResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/auth/oauth/google/callback [post]
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	var req dto.GoogleCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	accessToken, refreshToken, user, err := h.authService.HandleGoogleCallback(c.Request.Context(), req.Code)
	if err != nil {
		respondError(c, err, "Failed to authenticate with Google")
		return
	}

	utils.SetAuthCookies(
		c, h.cfg,
		utils.UserAccessCookie, utils.UserRefreshCookie,
		accessToken, refreshToken,
		h.cfg.JWTAccessExpiry, h.cfg.JWTRefreshExpiry,
	)

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Login successful", dto.NewAuthResponse(user)))
}

// RefreshToken handles POST /api/auth/refresh
//
//	@Summary		Refresh session
//	@Description	Rotates the refresh token cookie and issues a new access token.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	dto.SuccessResponse{data=dto.AuthResponse}
//	@Failure		401	{object}	dto.ErrorResponse
//	@Router			/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	rawRefreshToken, _ := c.Cookie(utils.UserRefreshCookie)

	accessToken, refreshToken, user, err := h.authService.RefreshToken(c.Request.Context(), rawRefreshToken)
	if err != nil {
		utils.ClearAuthCookies(c, h.cfg, utils.UserAccessCookie, utils.UserRefreshCookie)
		respondError(c, err, "Failed to refresh session")
		return
	}

	utils.SetAuthCookies(
		c, h.cfg,
		utils.UserAccessCookie, utils.UserRefreshCookie,
		accessToken, refreshToken,
		h.cfg.JWTAccessExpiry, h.cfg.JWTRefreshExpiry,
	)

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewAuthResponse(user)))
}

// GetCurrentUser handles GET /api/auth/me
//
//	@Summary		Get current user
//	@Description	Returns the profile of the currently authenticated user.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	dto.SuccessResponse{data=dto.UserResponse}
//	@Failure		401	{object}	dto.ErrorResponse
//	@Router			/auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Missing authenticated user", nil))
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err, "Failed to fetch current user")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewUserResponse(user)))
}

// Logout handles POST /api/auth/logout
//
//	@Summary		Logout
//	@Description	Revokes the current refresh token and clears session cookies.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	dto.SuccessResponse
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	rawRefreshToken, _ := c.Cookie(utils.UserRefreshCookie)

	if err := h.authService.Logout(c.Request.Context(), rawRefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("Failed to logout", err))
		return
	}

	utils.ClearAuthCookies(c, h.cfg, utils.UserAccessCookie, utils.UserRefreshCookie)
	c.JSON(http.StatusOK, dto.NewSuccessResponse("Logout successful", nil))
}

// respondError translates a service error into a standardized JSON error response.
func respondError(c *gin.Context, err error, fallbackMessage string) {
	var appErr *utils.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.Code, dto.NewErrorResponse(appErr.Message, appErr.Err))
		return
	}

	switch {
	case errors.Is(err, utils.ErrNotFound):
		c.JSON(http.StatusNotFound, dto.NewErrorResponse(fallbackMessage, err))
	case errors.Is(err, utils.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(fallbackMessage, err))
	case errors.Is(err, utils.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(fallbackMessage, err))
	case errors.Is(err, utils.ErrForbidden):
		c.JSON(http.StatusForbidden, dto.NewErrorResponse(fallbackMessage, err))
	case errors.Is(err, utils.ErrConflict):
		c.JSON(http.StatusConflict, dto.NewErrorResponse(fallbackMessage, err))
	default:
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse(fallbackMessage, err))
	}
}
