package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var unsafeCSRFMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// CSRFMiddleware implements the double-submit cookie pattern. Auth cookies
// are SameSite=None (cross-site, since frontend/admin/backend all live on
// different domains), which disables the browser's default CSRF mitigation -
// so state-changing requests must also echo the CSRF cookie's value back as
// an X-CSRF-Token header. A cross-site attacker can make the browser send the
// cookie, but can't read its value to put in the header.
func CSRFMiddleware(csrfCookie string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !unsafeCSRFMethods[c.Request.Method] {
			c.Next()
			return
		}

		cookieValue, err := c.Cookie(csrfCookie)
		headerValue := c.GetHeader("X-CSRF-Token")

		if err != nil || cookieValue == "" || headerValue == "" || cookieValue != headerValue {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Invalid or missing CSRF token",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
