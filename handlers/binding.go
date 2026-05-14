package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var DB *gorm.DB

func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "Некорректное тело JSON-запроса")
		return false
	}
	return true
}

func respondOK(c *gin.Context, payload any) {
	c.JSON(http.StatusOK, payload)
}

func respondCreated(c *gin.Context, payload any) {
	c.JSON(http.StatusCreated, payload)
}

func respondNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
