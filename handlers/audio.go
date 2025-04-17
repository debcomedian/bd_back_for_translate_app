package handlers

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

var SttClient *STTClient

func UploadDataHandler(c *gin.Context) {

	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		log.Printf("Ошибка получения аудиофайла: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не удалось получить аудиофайл"})
		return
	}
	defer file.Close()

	tempFilePath := filepath.Join(os.TempDir(), header.Filename)
	out, err := os.Create(tempFilePath)
	if err != nil {
		log.Printf("Ошибка создания временного файла: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения файла"})
		return
	}
	defer out.Close()

	if _, err = io.Copy(out, file); err != nil {
		log.Printf("Ошибка копирования данных в файл: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка записи файла"})
		return
	}

	result, err := SttClient.Process(tempFilePath)
	if err != nil {
		log.Printf("Ошибка обработки файла через Python процесс: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := os.Remove(tempFilePath); err != nil {
		log.Printf("Ошибка удаления временного файла: %v", err)
	} else {
		log.Printf("Временный файл успешно удалён: %s", tempFilePath)
	}

	c.JSON(http.StatusOK, gin.H{"alignment": result})
}
