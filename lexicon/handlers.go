package lexicon

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct{ DB *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{DB: db} }

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "scope": "lexicon"})
}

func (h *Handler) GetSnapshot(c *gin.Context) {
	resp, err := LoadSnapshot(h.DB)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ImportActiveBank(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}
	_ = c.ShouldBindJSON(&req)
	if strings.TrimSpace(req.Path) == "" {
		req.Path = os.Getenv("ACTIVE_BANK_PATH")
	}
	if strings.TrimSpace(req.Path) == "" {
		req.Path = "data/lexicon/active_bank.jsonl"
	}
	report, err := ImportActiveBank(h.DB, req.Path)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *Handler) RecalculateDirections(c *gin.Context) {
	report, err := RecalculateDirectionMeta(h.DB)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *Handler) GetCategories(c *gin.Context) {
	var items []Category
	if err := h.DB.Order("id ASC").Find(&items).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

type categoryRequest struct {
	Slug   string `json:"slug"`
	NameRu string `json:"name_ru"`
	NameEn string `json:"name_en"`
	NameDe string `json:"name_de"`
	Entity string `json:"entity"`
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "Некорректное тело JSON-запроса")
		return
	}
	item, err := buildCategoryFromRequest(req)
	if err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := h.DB.Create(&item).Error; err != nil {
		respondError(c, http.StatusConflict, "create_failed", err.Error())
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var existing Category
	if err := h.DB.First(&existing, id).Error; err != nil {
		respondError(c, http.StatusNotFound, "not_found", "Категория не найдена")
		return
	}
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "Некорректное тело JSON-запроса")
		return
	}
	item, err := buildCategoryFromRequest(req)
	if err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	updates := map[string]any{
		"slug":    item.Slug,
		"name_ru": item.NameRu,
		"name_en": item.NameEn,
		"name_de": item.NameDe,
		"entity":  item.Entity,
	}
	if err := h.DB.Model(&existing).Updates(updates).Error; err != nil {
		respondError(c, http.StatusConflict, "update_failed", err.Error())
		return
	}
	h.DB.First(&existing, id)
	c.JSON(http.StatusOK, existing)
}

func buildCategoryFromRequest(req categoryRequest) (Category, error) {
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	nameRu := strings.TrimSpace(req.NameRu)
	nameEn := strings.TrimSpace(req.NameEn)
	nameDe := strings.TrimSpace(req.NameDe)
	entity := strings.TrimSpace(req.Entity)
	if entity == "" {
		entity = "concept"
	}
	if slug == "" {
		return Category{}, fmt.Errorf("код категории не должен быть пустым")
	}
	if containsDigit(slug) {
		return Category{}, fmt.Errorf("код категории не должен содержать цифры")
	}
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || r == '_' || r == '-' {
			continue
		}
		return Category{}, fmt.Errorf("код категории может содержать только латинские буквы, дефис и подчёркивание")
	}
	for label, value := range map[string]string{
		"Название RU": nameRu,
		"Название EN": nameEn,
		"Название DE": nameDe,
	} {
		if value == "" {
			return Category{}, fmt.Errorf("%s не должно быть пустым", label)
		}
		if containsDigit(value) {
			return Category{}, fmt.Errorf("%s не должно содержать цифры", label)
		}
	}
	if entity != "concept" && entity != "word" {
		return Category{}, fmt.Errorf("недопустимый тип сущности")
	}
	return Category{Slug: slug, NameRu: nameRu, NameEn: nameEn, NameDe: nameDe, Entity: entity}, nil
}

func containsDigit(value string) bool {
	for _, r := range value {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func (h *Handler) GetConcepts(c *gin.Context) {
	var items []LexicalConcept
	query := h.DB.Preload("Category").Preload("Meta").Preload("Forms").Where("is_active = TRUE")
	if err := query.Order("id ASC").Find(&items).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) GetForms(c *gin.Context) {
	var items []LexicalForm
	query := h.DB.Preload("Meta").Preload("Synonyms")
	if conceptID := strings.TrimSpace(c.Query("concept_id")); conceptID != "" {
		query = query.Where("concept_id = ?", conceptID)
	}
	if lang := strings.TrimSpace(c.Query("lang_code")); lang != "" {
		query = query.Where("lang_code = ?", lang)
	}
	if err := query.Order("concept_id ASC, lang_code ASC, id ASC").Find(&items).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) GetDirections(c *gin.Context) {
	resp, err := ListDirections(h.DB, DirectionListQuery{
		Page:           parseIntQuery(c, "page", 1),
		PageSize:       parseIntQuery(c, "page_size", 100),
		Search:         c.Query("q"),
		DirectionCode:  c.Query("direction_code"),
		SourceLangCode: c.Query("source_lang_code"),
		TargetLangCode: c.Query("target_lang_code"),
		CategoryID:     parseUintQuery(c, "category_id", 0),
		Unassigned:     c.Query("unassigned"),
		Active:         c.Query("active"),
		SortBy:         c.Query("sort_by"),
		SortDir:        c.Query("sort_dir"),
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetProgress(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var items []UserDirectionProgress
	if err := h.DB.Where("user_id = ?", uid).Order("direction_id ASC").Find(&items).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

func parseIntQuery(c *gin.Context, name string, fallback int) int {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseUintQuery(c *gin.Context, name string, fallback uint64) uint64 {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": code, "message": message})
}

func parseUintParam(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		respondError(c, http.StatusBadRequest, "bad_id", "Некорректный идентификатор: "+name)
		return 0, false
	}
	return id, true
}

func getUserID(c *gin.Context) (uint64, bool) {
	value, ok := c.Get("uid")
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized", "Идентификатор пользователя отсутствует")
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return uint64(v), true
	case uint:
		return uint64(v), true
	case uint64:
		return v, true
	case float64:
		return uint64(v), true
	default:
		respondError(c, http.StatusUnauthorized, "unauthorized", "Некорректный идентификатор пользователя")
		return 0, false
	}
}
