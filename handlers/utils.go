package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
