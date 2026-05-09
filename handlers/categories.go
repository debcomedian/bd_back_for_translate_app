package handlers

import (
	"net/http"
	"strings"

	"bd_back_for_translate_app/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type categoryRequest struct {
	Slug   string `json:"slug" binding:"required"`
	NameRu string `json:"name_ru" binding:"required"`
	NameEn string `json:"name_en" binding:"required"`
	NameDe string `json:"name_de" binding:"required"`
	Entity string `json:"entity"`
}

func GetCategories(c *gin.Context) {
	var list []models.Category
	if err := DB.Order("id ASC").Find(&list).Error; err != nil {
		handleDBErr(c, err)
		return
	}
	respondOK(c, list)
}

func CreateCategory(c *gin.Context) {
	var req categoryRequest
	if !bindJSON(c, &req) {
		return
	}

	req.Slug = strings.TrimSpace(req.Slug)
	req.NameRu = strings.TrimSpace(req.NameRu)
	req.NameEn = strings.TrimSpace(req.NameEn)
	req.NameDe = strings.TrimSpace(req.NameDe)
	req.Entity = strings.TrimSpace(req.Entity)
	if req.Entity == "" {
		req.Entity = "word"
	}
	if req.Slug == "" || req.NameRu == "" || req.NameEn == "" || req.NameDe == "" {
		writeError(c, http.StatusUnprocessableEntity, "validation_error", "slug, name_ru, name_en and name_de are required")
		return
	}

	obj := models.Category{
		Slug:   req.Slug,
		NameRu: req.NameRu,
		NameEn: req.NameEn,
		NameDe: req.NameDe,
		Entity: req.Entity,
	}

	if handleDBErr(c, DB.Create(&obj).Error) {
		return
	}
	respondCreated(c, obj)
}

func UpdateCategory(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	var obj models.Category
	if err := DB.First(&obj, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			writeError(c, http.StatusNotFound, "not_found", "category not found")
			return
		}
		handleDBErr(c, err)
		return
	}

	var req categoryRequest
	if !bindJSON(c, &req) {
		return
	}

	updates := map[string]any{
		"slug":    strings.TrimSpace(req.Slug),
		"name_ru": strings.TrimSpace(req.NameRu),
		"name_en": strings.TrimSpace(req.NameEn),
		"name_de": strings.TrimSpace(req.NameDe),
		"entity":  strings.TrimSpace(req.Entity),
	}
	if updates["entity"] == "" {
		updates["entity"] = "word"
	}

	if handleDBErr(c, DB.Model(&obj).Updates(updates).Error) {
		return
	}
	if err := DB.First(&obj, id).Error; err != nil {
		handleDBErr(c, err)
		return
	}
	respondOK(c, obj)
}
