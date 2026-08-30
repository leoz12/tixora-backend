package middleware

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

var unsafeCSRFMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// CSRFMiddleware protects cookie-authenticated, state-changing requests by
// checking their origin against an allowlist.
//
// The auth cookies are SameSite=None (frontend/admin/backend live on separate
// domains), which disables the browser's built-in CSRF mitigation. The old
// double-submit token can't work here either: the SPA lives on a different
// registrable domain than the API, so its JavaScript can never read the CSRF
// cookie to echo it back in a header.
//
// Instead we rely on the fact that browsers always attach an Origin header to
// cross-origin state-changing requests (and to same-origin POST/PUT/PATCH/
// DELETE) and never let page scripts forge it. A request whose Origin (or, as
// a fallback, Referer) isn't in allowedOrigins is rejected. Requests with
// neither header are treated as non-browser clients (curl, server-to-server) -
// those can't be tricked into riding a victim's cookie, so they're not a CSRF
// vector. Every route using this middleware also sits behind an auth
// middleware, so an unauthenticated request never reaches here anyway.
func CSRFMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = true
	}

	return func(c *gin.Context) {
		if !unsafeCSRFMethods[c.Request.Method] {
			c.Next()
			return
		}

		if origin := c.GetHeader("Origin"); origin != "" {
			if !allowed[origin] {
				rejectCSRF(c)
				return
			}
			c.Next()
			return
		}

		// No Origin header: fall back to the Referer's scheme+host.
		if referer := c.GetHeader("Referer"); referer != "" {
			u, err := url.Parse(referer)
			if err != nil || !allowed[u.Scheme+"://"+u.Host] {
				rejectCSRF(c)
				return
			}
			c.Next()
			return
		}

		// Neither Origin nor Referer - not a browser-driven cross-site
		// request, so not a CSRF vector.
		c.Next()
	}
}

func rejectCSRF(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"message": "Cross-site request blocked",
	})
	c.Abort()
}
