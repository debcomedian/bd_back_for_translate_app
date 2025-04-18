package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Text — модель текста с внешним ключом на Category
// Ассоциация подтягивается через Preload в хендлерах
type Text struct {
	ID              int    `gorm:"primaryKey;column:id"        json:"id"`
	TitleRu         string `gorm:"column:title_ru"             json:"title_ru"`
	TitleEn         string `gorm:"column:title_en"             json:"title_en"`
	TitleDe         string `gorm:"column:title_de"             json:"title_de"`
	ContentRu       string `gorm:"column:content_ru"           json:"content_ru"`
	ContentEn       string `gorm:"column:content_en"           json:"content_en"`
	ContentDe       string `gorm:"column:content_de"           json:"content_de"`
	TranscriptionRu string `gorm:"column:transcription_ru"     json:"transcription_ru"`
	TranscriptionEn string `gorm:"column:transcription_en"     json:"transcription_en"`
	TranscriptionDe string `gorm:"column:transcription_de"     json:"transcription_de"`
	AudioRu         []byte `gorm:"column:audio_ru"             json:"audio_ru"`
	AudioEn         []byte `gorm:"column:audio_en"             json:"audio_en"`
	AudioDe         []byte `gorm:"column:audio_de"             json:"audio_de"`
	CategoryID      int    `gorm:"column:category_id"          json:"category_id"`
}

func (Text) TableName() string { return "texts" }

func GetTexts(c *gin.Context) {
	var list []Text
	if err := DB.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func CreateText(c *gin.Context) {
	var obj Text
	if !bindJSON(c, &obj) {
		return
	}
	obj.AudioRu, obj.AudioEn, obj.AudioDe = genAudioForText(&obj)
	if err := DB.Create(&obj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	DB.First(&obj, obj.ID)
	c.JSON(http.StatusCreated, obj)
}

func UpdateText(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	var obj Text
	if err := DB.First(&obj, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "text not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	var input Text
	if !bindJSON(c, &input) {
		return
	}
	obj = Text{
		ID:              id,
		TitleRu:         input.TitleRu,
		TitleEn:         input.TitleEn,
		TitleDe:         input.TitleDe,
		ContentRu:       input.ContentRu,
		ContentEn:       input.ContentEn,
		ContentDe:       input.ContentDe,
		TranscriptionRu: input.TranscriptionRu,
		TranscriptionEn: input.TranscriptionEn,
		TranscriptionDe: input.TranscriptionDe,
		CategoryID:      input.CategoryID,
	}
	obj.AudioRu, obj.AudioEn, obj.AudioDe = genAudioForText(&obj)
	if err := DB.Save(&obj).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	DB.First(&obj, id)
	c.JSON(http.StatusOK, obj)
}

func DeleteText(c *gin.Context) {
	id, ok := getID(c)
	if !ok {
		return
	}
	if err := DB.Delete(&Text{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
