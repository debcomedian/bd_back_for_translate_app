package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgconn"
)

func getID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

func bindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func genAudioForWord(w *Word) (ru, en, de []byte) {
		return nil, nil, nil
}

func genAudioForText(t *Text) (ru, en, de []byte) {
		return nil, nil, nil
}

func handleDBErr(c *gin.Context, err error) bool {
    if err == nil {
        return false
    }

    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        if info, ok := sqlStateMap[pgErr.Code]; ok {
            c.JSON(info.Status, gin.H{"error": info.Message})
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"error": pgErr.Message})
        }
        return true
    }

    msg := err.Error()
    for code, info := range sqlStateMap {
        if strings.Contains(msg, code) {
            c.JSON(info.Status, gin.H{"error": info.Message})
            return true
        }
    }

    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return true
}