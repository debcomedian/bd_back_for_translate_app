package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Middleware проверяет Bearer JWT и, при необходимости, роль.
//
// Поддерживаемые режимы:
//   - Middleware("")        -> любой авторизованный пользователь
//   - Middleware("editor")  -> editor или admin
//   - Middleware("admin")   -> только admin
//   - Middleware("<role>")  -> точное совпадение роли
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

		if !roleAllowed(requiredRole, claims.Role) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Set("uid", claims.UID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func roleAllowed(requiredRole, actualRole string) bool {
	switch requiredRole {
	case "":
		return actualRole != ""
	case "admin":
		return actualRole == "admin"
	case "editor":
		return actualRole == "editor" || actualRole == "admin"
	default:
		return actualRole == requiredRole
	}
}
