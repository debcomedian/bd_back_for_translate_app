package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Middleware("editor") – пускает editor + admin.
// Middleware("admin")  – пускает только admin.
func Middleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		claims, err := Parse(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// проверка роли
		if requiredRole == "admin" && claims.Role != "admin" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Set("uid", claims.UID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
