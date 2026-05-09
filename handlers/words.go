package handlers

import (
	"net/http"
	"strings"

	"bd_back_for_translate_app/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type wordRequest struct {
	LangCode        string  `json:"lang_code" binding:"required"`
	WordRu          *string `json:"word_ru"`
	WordEn          *string `json:"word_en"`
	WordDe          *string `json:"word_de"`
	TranscriptionRu *string `json:"transcription_ru"`
	TranscriptionEn *string `json:"transcription_en"`
	TranscriptionDe *string `json:"transcription_de"`
	SourceRef       *string `json:"source_ref"`
	CategoryID      *uint64 `json:"category_id"`
	IsActive        *bool   `json:"is_active"`
}

func GetWords(c *gin.Context) {
	q := DB.Model(&models.Word{}).Preload("Category").Preload("MetaBase")

	if lang := strings.TrimSpace(c.Query("lang_code")); lang != "" {
		q = q.Where("lang_code = ?", lang)
	}
	if active := strings.TrimSpace(c.Query("is_active")); active == "true" {
		q = q.Where("is_active = TRUE")
	} else if active == "false" {
		q = q.Where("is_active = FALSE")
	}
	if categoryID := strings.TrimSpace(c.Query("category_id")); categoryID != "" {
		q = q.Where("category_id = ?", categoryID)
	}

	var list []models.Word
	if err := q.Order("id ASC").Find(&list).Error; err != nil {
		handleDBErr(c, err)
		return
	}
	respondOK(c, list)
}

func CreateWord(c *gin.Context) {
	var req wordRequest
	if !bindJSON(c, &req) {
		return
	}
	if msg := validateWordRequest(req); msg != "" {
		writeError(c, http.StatusUnprocessableEntity, "validation_error", msg)
		return
	}

	obj := models.Word{
		LangCode:        strings.TrimSpace(req.LangCode),
		WordRu:          trimStringPtr(req.WordRu),
		WordEn:          trimStringPtr(req.WordEn),
		WordDe:          trimStringPtr(req.WordDe),
		TranscriptionRu: trimStringPtr(req.TranscriptionRu),
		TranscriptionEn: trimStringPtr(req.TranscriptionEn),
		TranscriptionDe: trimStringPtr(req.TranscriptionDe),
		SourceRef:       trimStringPtr(req.SourceRef),
		CategoryID:      req.CategoryID,
		IsActive:        req.IsActive == nil || *req.IsActive,
	}

	if handleDBErr(c, DB.Create(&obj).Error) {
		return
	}
	if err := DB.Preload("Category").Preload("MetaBase").First(&obj, obj.ID).Error; err != nil {
		handleDBErr(c, err)
		return
	}
	respondCreated(c, obj)
}

func UpdateWord(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	var obj models.Word
	if err := DB.First(&obj, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			writeError(c, http.StatusNotFound, "not_found", "word not found")
			return
		}
		handleDBErr(c, err)
		return
	}

	var req wordRequest
	if !bindJSON(c, &req) {
		return
	}
	if msg := validateWordRequest(req); msg != "" {
		writeError(c, http.StatusUnprocessableEntity, "validation_error", msg)
		return
	}

	updates := map[string]any{
		"lang_code":        strings.TrimSpace(req.LangCode),
		"word_ru":          trimStringPtr(req.WordRu),
		"word_en":          trimStringPtr(req.WordEn),
		"word_de":          trimStringPtr(req.WordDe),
		"transcription_ru": trimStringPtr(req.TranscriptionRu),
		"transcription_en": trimStringPtr(req.TranscriptionEn),
		"transcription_de": trimStringPtr(req.TranscriptionDe),
		"source_ref":       trimStringPtr(req.SourceRef),
		"category_id":      req.CategoryID,
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if handleDBErr(c, DB.Model(&obj).Updates(updates).Error) {
		return
	}
	if err := DB.Preload("Category").Preload("MetaBase").First(&obj, id).Error; err != nil {
		handleDBErr(c, err)
		return
	}
	respondOK(c, obj)
}

func validateWordRequest(req wordRequest) string {
	lang := strings.TrimSpace(req.LangCode)
	switch lang {
	case "ru":
		if req.WordRu == nil || strings.TrimSpace(*req.WordRu) == "" {
			return "word_ru is required for lang_code=ru"
		}
	case "en":
		if req.WordEn == nil || strings.TrimSpace(*req.WordEn) == "" {
			return "word_en is required for lang_code=en"
		}
	case "de":
		if req.WordDe == nil || strings.TrimSpace(*req.WordDe) == "" {
			return "word_de is required for lang_code=de"
		}
	default:
		return "lang_code must be one of: ru, en, de"
	}
	return ""
}

func trimStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
