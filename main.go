package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Word struct {
	ID            int    `json:"id"`
	Word          string `json:"word"`
	TranslationEn string `json:"translation_en"`
	TranslationRu string `json:"translation_ru"`
	TranslationDe string `json:"translation_de"`
	Category      string `json:"category"`
	Type          string `json:"type"`
	Status        string `json:"status"`
}

type ReadingText struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	Category     string `json:"category"`
	Translation1 string `json:"translation_variant1"`
	Translation2 string `json:"translation_variant2"`
}

var dbPool *pgxpool.Pool

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	var err error
	dbPool, err = pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	router := gin.Default()

	// Маршруты для работы со словами.
	router.GET("/api/words", getWords)
	router.GET("/api/words/type/:type", getWordsByType)
	router.POST("/api/words", createWord)
	router.PUT("/api/words/:id", updateWord)
	router.DELETE("/api/words/:id", deleteWord)

	// Маршруты для работы с текстами.
	router.GET("/api/texts/small", getTextsHandler("texts_small"))
	router.POST("/api/texts/small", createTextHandler("texts_small"))
	router.PUT("/api/texts/small/:id", updateTextHandler("texts_small"))
	router.DELETE("/api/texts/small/:id", deleteTextHandler("texts_small"))

	router.GET("/api/texts/medium", getTextsHandler("texts_medium"))
	router.POST("/api/texts/medium", createTextHandler("texts_medium"))
	router.PUT("/api/texts/medium/:id", updateTextHandler("texts_medium"))
	router.DELETE("/api/texts/medium/:id", deleteTextHandler("texts_medium"))

	router.GET("/api/texts/large", getTextsHandler("texts_large"))
	router.POST("/api/texts/large", createTextHandler("texts_large"))
	router.PUT("/api/texts/large/:id", updateTextHandler("texts_large"))
	router.DELETE("/api/texts/large/:id", deleteTextHandler("texts_large"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

// ************** Обработчики для слов **************
func getWords(c *gin.Context) {
	rows, err := dbPool.Query(context.Background(),
		"SELECT id, word, translation_en, translation_ru, translation_de, category, type, status FROM words")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	words := []Word{}
	for rows.Next() {
		var w Word
		if err := rows.Scan(&w.ID, &w.Word, &w.TranslationEn, &w.TranslationRu, &w.TranslationDe, &w.Category, &w.Type, &w.Status); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		words = append(words, w)
	}
	c.JSON(http.StatusOK, words)
}

func getWordsByType(c *gin.Context) {
	wordType := c.Param("type")
	rows, err := dbPool.Query(context.Background(),
		"SELECT id, word, translation_en, translation_ru, translation_de, category, type, status FROM words WHERE type=$1", wordType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	words := []Word{}
	for rows.Next() {
		var w Word
		if err := rows.Scan(&w.ID, &w.Word, &w.TranslationEn, &w.TranslationRu, &w.TranslationDe, &w.Category, &w.Type, &w.Status); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		words = append(words, w)
	}
	c.JSON(http.StatusOK, words)
}

func createWord(c *gin.Context) {
	var newWord Word
	if err := c.ShouldBindJSON(&newWord); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := dbPool.QueryRow(context.Background(),
		"INSERT INTO words (word, translation_en, translation_ru, translation_de, category, type, status) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		newWord.Word, newWord.TranslationEn, newWord.TranslationRu, newWord.TranslationDe, newWord.Category, newWord.Type, newWord.Status,
	).Scan(&newWord.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, newWord)
}

func updateWord(c *gin.Context) {
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
	cmdTag, err := dbPool.Exec(context.Background(),
		"UPDATE words SET word=$1, translation_en=$2, translation_ru=$3, translation_de=$4, category=$5, type=$6, status=$7 WHERE id=$8",
		updatedWord.Word, updatedWord.TranslationEn, updatedWord.TranslationRu, updatedWord.TranslationDe, updatedWord.Category, updatedWord.Type, updatedWord.Status, updatedWord.ID,
	)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}
	c.JSON(http.StatusOK, updatedWord)
}

func deleteWord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	cmdTag, err := dbPool.Exec(context.Background(), "DELETE FROM words WHERE id=$1", id)
	if err != nil || cmdTag.RowsAffected() == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
		return
	}
	c.Status(http.StatusNoContent)
}

// ************** Обработчики для текстов **************
func getTextsHandler(tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := "SELECT id, title, content, category, translation_variant1, translation_variant2 FROM " + tableName
		rows, err := dbPool.Query(context.Background(), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		texts := []ReadingText{}
		for rows.Next() {
			var rt ReadingText
			if err := rows.Scan(&rt.ID, &rt.Title, &rt.Content, &rt.Category, &rt.Translation1, &rt.Translation2); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			texts = append(texts, rt)
		}
		c.JSON(http.StatusOK, texts)
	}
}

func createTextHandler(tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newText ReadingText
		if err := c.ShouldBindJSON(&newText); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		query := "INSERT INTO " + tableName + " (title, content, category, translation_variant1, translation_variant2) VALUES ($1, $2, $3, $4, $5) RETURNING id"
		err := dbPool.QueryRow(context.Background(), query,
			newText.Title, newText.Content, newText.Category, newText.Translation1, newText.Translation2,
		).Scan(&newText.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, newText)
	}
}

func updateTextHandler(tableName string) gin.HandlerFunc {
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
		query := "UPDATE " + tableName + " SET title=$1, content=$2, category=$3, translation_variant1=$4, translation_variant2=$5 WHERE id=$6"
		cmdTag, err := dbPool.Exec(context.Background(), query,
			updatedText.Title, updatedText.Content, updatedText.Category, updatedText.Translation1, updatedText.Translation2, updatedText.ID,
		)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
			return
		}
		c.JSON(http.StatusOK, updatedText)
	}
}

func deleteTextHandler(tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}
		query := "DELETE FROM " + tableName + " WHERE id=$1"
		cmdTag, err := dbPool.Exec(context.Background(), query, id)
		if err != nil || cmdTag.RowsAffected() == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
