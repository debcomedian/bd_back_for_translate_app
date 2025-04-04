package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"               // Импорт библиотеки для загрузки .env
	"github.com/jackc/pgx/v4/pgxpool"
)

// Word представляет модель слова
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

var dbPool *pgxpool.Pool

func main() {
	// Загрузка переменных окружения из файла .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	// Настройка подключения к PostgreSQL
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

	// Инициализация Gin
	router := gin.Default()

	// Определяем маршруты API
	router.GET("/api/words", getWords)
	router.GET("/api/words/type/:type", getWordsByType)
	router.POST("/api/words", createWord)
	router.PUT("/api/words/:id", updateWord)
	router.DELETE("/api/words/:id", deleteWord)

	// Получаем порт из переменной окружения, по умолчанию 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

// getWords возвращает все слова
func getWords(c *gin.Context) {
	rows, err := dbPool.Query(context.Background(), "SELECT id, word, translation_en, translation_ru, translation_de, category, type, status FROM words")
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

// getWordsByType возвращает слова по типу (например, nouns, adjectives и т.д.)
func getWordsByType(c *gin.Context) {
	wordType := c.Param("type")
	rows, err := dbPool.Query(context.Background(), "SELECT id, word, translation_en, translation_ru, translation_de, category, type, status FROM words WHERE type=$1", wordType)
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

// createWord добавляет новое слово
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

// updateWord обновляет слово по ID
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

// deleteWord удаляет слово по ID
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
