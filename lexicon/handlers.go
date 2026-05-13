package lexicon

import (
	"net/http"
	"os"
	"strconv"
	"strings"

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

func (h *Handler) RebuildFromCurrent(c *gin.Context) {
	report, err := RebuildFromCurrent(h.DB)
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

func (h *Handler) CreateCategory(c *gin.Context) {
	var req Category
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	req.ID = 0
	if req.Entity == "" {
		req.Entity = "concept"
	}
	if err := h.DB.Create(&req).Error; err != nil {
		respondError(c, http.StatusConflict, "create_failed", err.Error())
		return
	}
	c.JSON(http.StatusCreated, req)
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var existing Category
	if err := h.DB.First(&existing, id).Error; err != nil {
		respondError(c, http.StatusNotFound, "not_found", err.Error())
		return
	}
	var req Category
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	updates := map[string]any{"slug": req.Slug, "name_ru": req.NameRu, "name_en": req.NameEn, "name_de": req.NameDe, "entity": req.Entity}
	if updates["entity"] == "" {
		updates["entity"] = "concept"
	}
	if err := h.DB.Model(&existing).Updates(updates).Error; err != nil {
		respondError(c, http.StatusConflict, "update_failed", err.Error())
		return
	}
	h.DB.First(&existing, id)
	c.JSON(http.StatusOK, existing)
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
	resp, err := ListDirections(h.DB)
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

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": code, "message": message})
}

func parseUintParam(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		respondError(c, http.StatusBadRequest, "bad_id", "invalid "+name)
		return 0, false
	}
	return id, true
}

func getUserID(c *gin.Context) (uint64, bool) {
	value, ok := c.Get("uid")
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized", "missing user id")
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
		respondError(c, http.StatusUnauthorized, "unauthorized", "invalid user id")
		return 0, false
	}
}
