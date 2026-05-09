package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func parseID(c *gin.Context, name string) (uint64, bool) {
	raw := c.Param(name)
	if raw == "" {
		writeError(c, http.StatusBadRequest, "invalid_id", fmt.Sprintf("path param %s is required", name))
		return 0, false
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_id", fmt.Sprintf("path param %s must be uint", name))
		return 0, false
	}
	return id, true
}

func getUserIDFromContext(c *gin.Context) (uint64, bool) {
	v, ok := c.Get("uid")
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return 0, false
	}

	switch x := v.(type) {
	case int:
		return uint64(x), true
	case int64:
		return uint64(x), true
	case uint64:
		return x, true
	case uint:
		return uint64(x), true
	case float64:
		return uint64(x), true
	default:
		writeError(c, http.StatusUnauthorized, "unauthorized", "invalid auth context")
		return 0, false
	}
}
