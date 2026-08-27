package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tixora/internal/utils"
)

// AuthMiddleware verifies the access token cookie on incoming requests and
// stores the authenticated user's ID and email in the request context.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie(utils.UserAccessCookie)
		if err != nil || tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Missing or expired session",
			})
			c.Abort()
			return
		}

		claims, err := utils.VerifyJWT(tokenString, jwtSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}
