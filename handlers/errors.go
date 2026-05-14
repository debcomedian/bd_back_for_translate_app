package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgconn"
)

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

var sqlStateMap = map[string]struct {
	Status  int
	Message string
}{
	"23502": {Status: http.StatusUnprocessableEntity, Message: "Нарушение ограничения NOT NULL"},
	"23503": {Status: http.StatusBadRequest, Message: "Нарушение внешнего ключа"},
	"23505": {Status: http.StatusConflict, Message: "Дублирующее значение ключа"},
}

func writeError(c *gin.Context, status int, code string, message string) {
	c.JSON(status, errorResponse{Error: code, Message: message})
}

func handleDBErr(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if info, ok := sqlStateMap[pgErr.Code]; ok {
			writeError(c, info.Status, pgErr.Code, info.Message)
			return true
		}
		writeError(c, http.StatusBadRequest, pgErr.Code, pgErr.Message)
		return true
	}

	msg := err.Error()
	for code, info := range sqlStateMap {
		if strings.Contains(msg, code) {
			writeError(c, info.Status, code, info.Message)
			return true
		}
	}

	writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
	return true
}
