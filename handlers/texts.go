package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetTextsHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := `
			SELECT id,
			       title_ru, title_en, title_de,
			       content_ru, content_en, content_de,
			       transcription_ru, transcription_en, transcription_de,
			       audio_ru, audio_en, audio_de,
			       category_id
			FROM ` + table
		rows, err := Dbpool.Query(context.Background(), q)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var list []ReadingText
		for rows.Next() {
			var t ReadingText
			if err := rows.Scan(
				&t.ID,
				&t.TitleRu, &t.TitleEn, &t.TitleDe,
				&t.ContentRu, &t.ContentEn, &t.ContentDe,
				&t.TranscriptionRu, &t.TranscriptionEn, &t.TranscriptionDe,
				&t.AudioRu, &t.AudioEn, &t.AudioDe,
				&t.CategoryID,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			list = append(list, t)
		}
		c.JSON(http.StatusOK, list)
	}
}

func CreateTextHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var t ReadingText
		if err := c.ShouldBindJSON(&t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		q := `
			INSERT INTO ` + table + ` (
				title_ru, title_en, title_de,
				content_ru, content_en, content_de,
				transcription_ru, transcription_en, transcription_de,
				audio_ru, audio_en, audio_de,
				category_id
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
			RETURNING id`
		err := Dbpool.QueryRow(context.Background(), q,
			t.TitleRu, t.TitleEn, t.TitleDe,
			t.ContentRu, t.ContentEn, t.ContentDe,
			t.TranscriptionRu, t.TranscriptionEn, t.TranscriptionDe,
			nil, nil, nil, // пока без аудио
			t.CategoryID,
		).Scan(&t.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		genAudioForText(table, &t)
		c.JSON(http.StatusCreated, t)
	}
}

func UpdateTextHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var t ReadingText
		if err := c.ShouldBindJSON(&t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		t.ID = id
		q := `
			UPDATE ` + table + ` SET
				title_ru=$1, title_en=$2, title_de=$3,
				content_ru=$4, content_en=$5, content_de=$6,
				transcription_ru=$7, transcription_en=$8, transcription_de=$9,
				audio_ru=$10, audio_en=$11, audio_de=$12,
				category_id=$13
			WHERE id=$14`
		cmd, err := Dbpool.Exec(context.Background(), q,
			t.TitleRu, t.TitleEn, t.TitleDe,
			t.ContentRu, t.ContentEn, t.ContentDe,
			t.TranscriptionRu, t.TranscriptionEn, t.TranscriptionDe,
			t.AudioRu, t.AudioEn, t.AudioDe,
			t.CategoryID, t.ID,
		)
		if err != nil || cmd.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
			return
		}
		genAudioForText(table, &t)
		c.JSON(http.StatusOK, t)
	}
}

func DeleteTextHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		cmd, err := Dbpool.Exec(context.Background(),
			"DELETE FROM "+table+" WHERE id=$1", id)
		if err != nil || cmd.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func genAudioForText(table string, t *ReadingText) {
	if TtsClient == nil {
		return
	}
	type rec struct {
		ipa  string
		code string
		ptr  *[]byte
		col  string
	}
	langs := []rec{
		{t.TranscriptionRu, "ru", &t.AudioRu, "audio_ru"},
		{t.TranscriptionEn, "en", &t.AudioEn, "audio_en"},
		{t.TranscriptionDe, "de", &t.AudioDe, "audio_de"},
	}
	for _, l := range langs {
		if l.ipa == "" {
			continue
		}
		if data, err := TtsClient.Synthesize(l.ipa, l.code); err == nil {
			*l.ptr = data
			_, _ = Dbpool.Exec(context.Background(),
				`UPDATE `+table+` SET `+l.col+`=$1 WHERE id=$2`, data, t.ID)
		}
	}
}
