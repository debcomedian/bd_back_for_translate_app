package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
)

var Dbpool *pgxpool.Pool

type Word struct {
	ID              int    `json:"id"`
	WordRu          string `json:"word_ru"`
	WordEn          string `json:"word_en"`
	WordDe          string `json:"word_de"`
	TranscriptionRu string `json:"transcription_ru"`
	TranscriptionEn string `json:"transcription_en"`
	TranscriptionDe string `json:"transcription_de"`
	AudioRu         []byte `json:"audio_ru"`
	AudioEn         []byte `json:"audio_en"`
	AudioDe         []byte `json:"audio_de"`
	CategoryID      int    `json:"category_id"`
	TypeRu          string `json:"type_ru"`
	TypeEn          string `json:"type_en"`
	TypeDe          string `json:"type_de"`
	Status          string `json:"status"`
}

func GetWords(c *gin.Context) {
	const q = `
		SELECT id, word_ru, word_en, word_de,
		       transcription_ru, transcription_en, transcription_de,
		       audio_ru, audio_en, audio_de,
		       category_id, type_ru, type_en, type_de, status
		FROM words`
	rows, err := Dbpool.Query(context.Background(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var out []Word
	for rows.Next() {
		var w Word
		if err := rows.Scan(
			&w.ID, &w.WordRu, &w.WordEn, &w.WordDe,
			&w.TranscriptionRu, &w.TranscriptionEn, &w.TranscriptionDe,
			&w.AudioRu, &w.AudioEn, &w.AudioDe,
			&w.CategoryID, &w.TypeRu, &w.TypeEn, &w.TypeDe,
			&w.Status,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, w)
	}
	c.JSON(http.StatusOK, out)
}

func GetWordsByType(c *gin.Context) {
	typ := c.Param("type")
	const q = `
		SELECT id, word_ru, word_en, word_de,
		       transcription_ru, transcription_en, transcription_de,
		       audio_ru, audio_en, audio_de,
		       category_id, type_ru, type_en, type_de, status
		FROM words WHERE type_ru = $1`
	rows, err := Dbpool.Query(context.Background(), q, typ)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var out []Word
	for rows.Next() {
		var w Word
		if err := rows.Scan(
			&w.ID, &w.WordRu, &w.WordEn, &w.WordDe,
			&w.TranscriptionRu, &w.TranscriptionEn, &w.TranscriptionDe,
			&w.AudioRu, &w.AudioEn, &w.AudioDe,
			&w.CategoryID, &w.TypeRu, &w.TypeEn, &w.TypeDe,
			&w.Status,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, w)
	}
	c.JSON(http.StatusOK, out)
}

func CreateWord(c *gin.Context) {
	var w Word
	if err := c.ShouldBindJSON(&w); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const ins = `
		INSERT INTO words (
			word_ru, word_en, word_de,
			transcription_ru, transcription_en, transcription_de,
			audio_ru, audio_en, audio_de,
			category_id, type_ru, type_en, type_de, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id`
	err := Dbpool.QueryRow(context.Background(), ins,
		w.WordRu, w.WordEn, w.WordDe,
		w.TranscriptionRu, w.TranscriptionEn, w.TranscriptionDe,
		nil, nil, nil,
		w.CategoryID, w.TypeRu, w.TypeEn, w.TypeDe, w.Status,
	).Scan(&w.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	genAudioForWord(&w)

	c.JSON(http.StatusCreated, w)
}

func UpdateWord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var w Word
	if err := c.ShouldBindJSON(&w); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	w.ID = id

	const upd = `
		UPDATE words SET
			word_ru=$1, word_en=$2, word_de=$3,
			transcription_ru=$4, transcription_en=$5, transcription_de=$6,
			audio_ru=$7, audio_en=$8, audio_de=$9,
			category_id=$10, type_ru=$11, type_en=$12, type_de=$13, status=$14
		WHERE id=$15`
	cmd, err := Dbpool.Exec(context.Background(), upd,
		w.WordRu, w.WordEn, w.WordDe,
		w.TranscriptionRu, w.TranscriptionEn, w.TranscriptionDe,
		w.AudioRu, w.AudioEn, w.AudioDe,
		w.CategoryID, w.TypeRu, w.TypeEn, w.TypeDe, w.Status,
		w.ID,
	)
	if err != nil || cmd.RowsAffected() == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	genAudioForWord(&w)

	c.JSON(http.StatusOK, w)
}

func DeleteWord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	cmd, err := Dbpool.Exec(context.Background(), `DELETE FROM words WHERE id=$1`, id)
	if err != nil || cmd.RowsAffected() == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.Status(http.StatusNoContent)
}

func genAudioForWord(w *Word) {
	if TtsClient == nil {
		return
	}
	type rec struct {
		text string // теперь само слово!
		lang string
		ptr  *[]byte
		col  string
	}
	langs := []rec{
		{w.WordRu, "ru", &w.AudioRu, "audio_ru"},
		{w.WordEn, "en", &w.AudioEn, "audio_en"},
		{w.WordDe, "de", &w.AudioDe, "audio_de"},
	}
	for _, r := range langs {
		if r.text == "" { // слова нет — пропускаем
			continue
		}
		wav, err := TtsClient.Synthesize(r.text, r.lang)
		if err != nil {
			log.Printf("[TTS] id=%d %s error: %v", w.ID, r.lang, err)
			continue
		}
		*r.ptr = wav
		_, _ = Dbpool.Exec(context.Background(),
			`UPDATE words SET `+r.col+`=$1 WHERE id=$2`, wav, w.ID)
	}
}
