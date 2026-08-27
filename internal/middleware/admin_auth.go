package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tixora/internal/utils"
)

// AdminAuthMiddleware verifies the admin access token cookie on incoming
// requests and stores the authenticated admin's ID, email, and role in the
// request context.
func AdminAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie(utils.AdminAccessCookie)
		if err != nil || tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Missing or expired session",
			})
			c.Abort()
			return
		}

		claims, err := utils.VerifyAdminJWT(tokenString, jwtSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		c.Set("admin_id", claims.AdminID)
		c.Set("admin_email", claims.Email)
		c.Set("admin_role", claims.Role)
		c.Next()
	}
}
