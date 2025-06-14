package handlers

import (
	"errors"
	"log"
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
	if TtsClient == nil {
		return nil, nil, nil
	}

	type rec struct {
		text string
		lang string
	}
	langs := []rec{
		{w.WordRu, "ru"},
		{w.WordEn, "en"},
		{w.WordDe, "de"},
	}

	for i, r := range langs {
		if r.text == "" {
			continue
		}
		wav, err := TtsClient.Synthesize(r.text, r.lang)
		if err != nil {
			log.Printf("[TTS] id=%d %s error: %v", w.ID, r.lang, err)
			continue
		}
		switch i {
		case 0:
			ru = wav
		case 1:
			en = wav
		case 2:
			de = wav
		}
	}
	return ru, en, de
}

func genAudioForText(t *Text) (ru, en, de []byte) {
	if TtsClient == nil {
		return nil, nil, nil
	}

	type rec struct {
		text string
		lang string
	}
	langs := []rec{
		{t.ContentRu, "ru"},
		{t.ContentEn, "en"},
		{t.ContentDe, "de"},
	}

	for i, r := range langs {
		if r.text == "" {
			continue
		}
		wav, err := TtsClient.Synthesize(r.text, r.lang)
		if err != nil {
			log.Printf("[TTS] id=%d %s error: %v", t.ID, r.lang, err)
			continue
		}
		switch i {
		case 0:
			ru = wav
		case 1:
			en = wav
		case 2:
			de = wav
		}
	}
	return ru, en, de
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