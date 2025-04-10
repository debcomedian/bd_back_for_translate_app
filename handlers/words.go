package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
)

var Dbpool *pgxpool.Pool

// ************** Обработчики для слов **************

func GetWords(c *gin.Context) {
	query := `
		SELECT id, word_ru, word_en, word_de,
		       category_id, type_ru, type_en, type_de, status 
		FROM words
	`
	rows, err := Dbpool.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	words := []Word{}
	for rows.Next() {
		var w Word
		if err := rows.Scan(
			&w.ID,
			&w.WordRu, &w.WordEn, &w.WordDe,
			&w.CategoryID,
			&w.TypeRu, &w.TypeEn, &w.TypeDe,
			&w.Status,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		words = append(words, w)
	}
	c.JSON(http.StatusOK, words)
}

func GetWordsByType(c *gin.Context) {
	wordType := c.Param("type")
	query := `
		SELECT id, word_ru, word_en, word_de,
		       category_id, type_ru, type_en, type_de, status 
		FROM words 
		WHERE type_ru = $1
	`
	rows, err := Dbpool.Query(context.Background(), query, wordType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	words := []Word{}
	for rows.Next() {
		var w Word
		if err := rows.Scan(
			&w.ID,
			&w.WordRu, &w.WordEn, &w.WordDe,
			&w.CategoryID,
			&w.TypeRu, &w.TypeEn, &w.TypeDe,
			&w.Status,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		words = append(words, w)
	}
	c.JSON(http.StatusOK, words)
}

func CreateWord(c *gin.Context) {
	var newWord Word
	if err := c.ShouldBindJSON(&newWord); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	query := `
		INSERT INTO words (
			word_ru, word_en, word_de, 
			category_id, type_ru, type_en, type_de, 
			status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	err := Dbpool.QueryRow(
		context.Background(),
		query,
		newWord.WordRu, newWord.WordEn, newWord.WordDe,
		newWord.CategoryID,
		newWord.TypeRu, newWord.TypeEn, newWord.TypeDe,
		newWord.Status,
	).Scan(&newWord.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, newWord)
}

func UpdateWord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var updatedWord Word
	if err := c.ShouldBindJSON(&updatedWord); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updatedWord.ID = id
	query := `
		UPDATE words 
		SET word_ru=$1, word_en=$2, word_de=$3, 
		    category_id=$4, type_ru=$5, type_en=$6, type_de=$7, 
		    status=$8 
		WHERE id=$9
	`
	cmdTag, err := Dbpool.Exec(
		context.Background(),
		query,
		updatedWord.WordRu, updatedWord.WordEn, updatedWord.WordDe,
		updatedWord.CategoryID,
		updatedWord.TypeRu, updatedWord.TypeEn, updatedWord.TypeDe,
		updatedWord.Status,
		updatedWord.ID,
	)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}
	c.JSON(http.StatusOK, updatedWord)
}

func DeleteWord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	cmdTag, err := Dbpool.Exec(context.Background(), "DELETE FROM words WHERE id=$1", id)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
		return
	}
	c.Status(http.StatusNoContent)
}
