package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"tixora/internal/config"
)

// Cookie names - user and admin sessions use separate names so a browser
// that's logged into both never mixes them up.
const (
	UserAccessCookie  = "tx_access_token"
	UserRefreshCookie = "tx_refresh_token"
	UserCSRFCookie    = "tx_csrf_token"

	AdminAccessCookie  = "tx_admin_access_token"
	AdminRefreshCookie = "tx_admin_refresh_token"
	AdminCSRFCookie    = "tx_admin_csrf_token"
)

// csrfTokenBytes is the entropy used for the double-submit CSRF cookie.
const csrfTokenBytes = 32

// SetAuthCookies sets the access, refresh, and CSRF cookies for a session and
// returns the raw CSRF token (handlers don't need it, but it's handy for
// tests). accessCookie/refreshCookie/csrfCookie pick which name triple to use
// - the User* or Admin* constants above.
func SetAuthCookies(
	c *gin.Context,
	cfg *config.Config,
	accessCookie, refreshCookie, csrfCookie string,
	accessToken, refreshToken string,
	accessExpiry, refreshExpiry time.Duration,
) (string, error) {
	csrfToken, err := GenerateRandomToken(csrfTokenBytes)
	if err != nil {
		return "", err
	}

	setCookie(c, cfg, accessCookie, accessToken, int(accessExpiry.Seconds()), true)
	setCookie(c, cfg, refreshCookie, refreshToken, int(refreshExpiry.Seconds()), true)
	setCookie(c, cfg, csrfCookie, csrfToken, int(refreshExpiry.Seconds()), false)

	return csrfToken, nil
}

// ClearAuthCookies expires the access/refresh/CSRF cookies (used on logout).
func ClearAuthCookies(c *gin.Context, cfg *config.Config, accessCookie, refreshCookie, csrfCookie string) {
	setCookie(c, cfg, accessCookie, "", -1, true)
	setCookie(c, cfg, refreshCookie, "", -1, true)
	setCookie(c, cfg, csrfCookie, "", -1, false)
}

// setCookie centralizes the flags every auth cookie needs: frontend, admin,
// and backend all live on different domains (Vercel/Railway), so cookies are
// cross-site and require SameSite=None + Secure. No Domain is set (host-only)
// since there's no shared parent domain to scope it to.
func setCookie(c *gin.Context, cfg *config.Config, name, value string, maxAgeSeconds int, httpOnly bool) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie(name, value, maxAgeSeconds, "/", "", cfg.CookieSecure, httpOnly)
}
