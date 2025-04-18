package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Word — модель слова с FK на Category
type Word struct {
	ID              int    `gorm:"primaryKey;column:id"      json:"id"`
	WordRu          string `gorm:"column:word_ru"            json:"word_ru"`
	WordEn          string `gorm:"column:word_en"            json:"word_en"`
	WordDe          string `gorm:"column:word_de"            json:"word_de"`
	TranscriptionRu string `gorm:"column:transcription_ru"   json:"transcription_ru"`
	TranscriptionEn string `gorm:"column:transcription_en"   json:"transcription_en"`
	TranscriptionDe string `gorm:"column:transcription_de"   json:"transcription_de"`
	AudioRu         []byte `gorm:"column:audio_ru"           json:"audio_ru"`
	AudioEn         []byte `gorm:"column:audio_en"           json:"audio_en"`
	AudioDe         []byte `gorm:"column:audio_de"           json:"audio_de"`
	CategoryID      int    `gorm:"column:category_id"        json:"category_id"`
	TypeRu          string `gorm:"column:type_ru"            json:"type_ru"`
	TypeEn          string `gorm:"column:type_en"            json:"type_en"`
	TypeDe          string `gorm:"column:type_de"            json:"type_de"`
	Status          string `gorm:"column:status"             json:"status"`
}

func (Word) TableName() string { return "words" }

func GetWords(c *gin.Context) {
	var list []Word
	if err := DB.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func CreateWord(c *gin.Context) {
	var obj Word
	if !bindJSON(c, &obj) {
		return
	}
	obj.AudioRu, obj.AudioEn, obj.AudioDe = genAudioForWord(&obj)
	if err := DB.Create(&obj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	DB.First(&obj, obj.ID)
	c.JSON(http.StatusCreated, obj)
}

func UpdateWord(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	var obj Word
	if err := DB.First(&obj, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "word not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	var input Word
	if !bindJSON(c, &input) {
		return
	}
	obj = Word{
		ID:              id,
		WordRu:          input.WordRu,
		WordEn:          input.WordEn,
		WordDe:          input.WordDe,
		TranscriptionRu: input.TranscriptionRu,
		TranscriptionEn: input.TranscriptionEn,
		TranscriptionDe: input.TranscriptionDe,
		CategoryID:      input.CategoryID,
		TypeRu:          input.TypeRu,
		TypeEn:          input.TypeEn,
		TypeDe:          input.TypeDe,
		Status:          input.Status,
	}
	obj.AudioRu, obj.AudioEn, obj.AudioDe = genAudioForWord(&obj)
	if err := DB.Save(&obj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	DB.First(&obj, id)
	c.JSON(http.StatusOK, obj)
}

func DeleteWord(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	if err := DB.Delete(&Word{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
