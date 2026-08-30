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

	AdminAccessCookie  = "tx_admin_access_token"
	AdminRefreshCookie = "tx_admin_refresh_token"
)

// SetAuthCookies sets the access and refresh cookies for a session.
// accessCookie/refreshCookie pick which name pair to use - the User* or
// Admin* constants above.
//
// There's no CSRF cookie: the frontend/admin SPA lives on a different
// registrable domain than the API, so its JavaScript can't read a
// double-submit cookie to echo it back. CSRF is enforced by an Origin/Referer
// allowlist instead (see internal/middleware/csrf.go).
func SetAuthCookies(
	c *gin.Context,
	cfg *config.Config,
	accessCookie, refreshCookie string,
	accessToken, refreshToken string,
	accessExpiry, refreshExpiry time.Duration,
) {
	setCookie(c, cfg, accessCookie, accessToken, int(accessExpiry.Seconds()), true)
	setCookie(c, cfg, refreshCookie, refreshToken, int(refreshExpiry.Seconds()), true)
}

// ClearAuthCookies expires the access/refresh cookies (used on logout).
func ClearAuthCookies(c *gin.Context, cfg *config.Config, accessCookie, refreshCookie string) {
	setCookie(c, cfg, accessCookie, "", -1, true)
	setCookie(c, cfg, refreshCookie, "", -1, true)
}

// setCookie centralizes the flags every auth cookie needs: frontend, admin,
// and backend all live on different domains (Vercel/Railway), so cookies are
// cross-site and require SameSite=None + Secure. No Domain is set (host-only)
// since there's no shared parent domain to scope it to.
func setCookie(c *gin.Context, cfg *config.Config, name, value string, maxAgeSeconds int, httpOnly bool) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie(name, value, maxAgeSeconds, "/", "", cfg.CookieSecure, httpOnly)
}
