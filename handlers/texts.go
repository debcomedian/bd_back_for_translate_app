package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ************** Обработчики для текстов **************

func GetTextsHandler(tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `
			SELECT id, 
			       title_ru, title_en, title_de, 
			       content_ru, content_en, content_de, 
			       category_id
			FROM ` + tableName
		rows, err := Dbpool.Query(context.Background(), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		texts := []ReadingText{}
		for rows.Next() {
			var rt ReadingText
			if err := rows.Scan(
				&rt.ID,
				&rt.TitleRu, &rt.TitleEn, &rt.TitleDe,
				&rt.ContentRu, &rt.ContentEn, &rt.ContentDe,
				&rt.CategoryID,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			texts = append(texts, rt)
		}
		c.JSON(http.StatusOK, texts)
	}
}

func CreateTextHandler(tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newText ReadingText
		if err := c.ShouldBindJSON(&newText); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		query := `
			INSERT INTO ` + tableName + ` (
				title_ru, title_en, title_de, 
				content_ru, content_en, content_de, 
				category_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`
		err := Dbpool.QueryRow(context.Background(), query,
			newText.TitleRu, newText.TitleEn, newText.TitleDe,
			newText.ContentRu, newText.ContentEn, newText.ContentDe,
			newText.CategoryID,
		).Scan(&newText.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, newText)
	}
}

func UpdateTextHandler(tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		var updatedText ReadingText
		if err := c.ShouldBindJSON(&updatedText); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updatedText.ID = id
		query := `
			UPDATE ` + tableName + ` 
			SET title_ru=$1, title_en=$2, title_de=$3, 
			    content_ru=$4, content_en=$5, content_de=$6, 
			    category_id=$7
			WHERE id=$8
		`
		cmdTag, err := Dbpool.Exec(context.Background(), query,
			updatedText.TitleRu, updatedText.TitleEn, updatedText.TitleDe,
			updatedText.ContentRu, updatedText.ContentEn, updatedText.ContentDe,
			updatedText.CategoryID,
			updatedText.ID,
		)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
		c.JSON(http.StatusOK, updatedText)
	}
}

func DeleteTextHandler(tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		query := "DELETE FROM " + tableName + " WHERE id=$1"
		cmdTag, err := Dbpool.Exec(context.Background(), query, id)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
