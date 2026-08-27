package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorMiddleware recovers from panics and converts any errors attached via
// c.Error(...) into a single JSON error response.
func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "Internal server error",
				})
				c.Abort()
			}
		}()

		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		err := c.Errors.Last()
		status := c.Writer.Status()
		if status < http.StatusBadRequest {
			status = http.StatusInternalServerError
		}

		c.JSON(status, gin.H{
			"success": false,
			"message": fmt.Sprint(err.Err),
		})
	}
}
