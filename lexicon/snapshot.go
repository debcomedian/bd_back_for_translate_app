package lexicon

import (
	"strings"

	"gorm.io/gorm"
)

type SnapshotResponse struct {
	SnapshotVersion *ContentSnapshotVersion `json:"snapshot_version,omitempty"`
	Languages       []Language              `json:"languages"`
	Categories      []Category              `json:"categories"`
	Concepts        []LexicalConcept        `json:"concepts"`
	Forms           []LexicalForm           `json:"forms"`
	ConceptMeta     []LexicalConceptMeta    `json:"concept_meta"`
	FormMeta        []LexicalFormMeta       `json:"form_meta"`
	FormSynonyms    []LexicalFormSynonym    `json:"form_synonyms"`
	Directions      []TrainingDirectionView `json:"directions"`
}

func LoadSnapshot(db *gorm.DB) (*SnapshotResponse, error) {
	resp := &SnapshotResponse{}

	var snapshot ContentSnapshotVersion
	err := db.Where("snapshot_type = ? AND is_active = TRUE", directionalSnapshotType).
		Order("version_code DESC").
		First(&snapshot).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if snapshot.ID != 0 {
		resp.SnapshotVersion = &snapshot
	}

	if err := db.Order("code ASC").Find(&resp.Languages).Error; err != nil {
		return nil, err
	}
	if err := db.Order("id ASC").Find(&resp.Categories).Error; err != nil {
		return nil, err
	}
	if err := db.Where("is_active = TRUE").Order("id ASC").Find(&resp.Concepts).Error; err != nil {
		return nil, err
	}
	if err := db.Order("concept_id ASC, lang_code ASC, id ASC").Find(&resp.Forms).Error; err != nil {
		return nil, err
	}
	if err := db.Order("concept_id ASC").Find(&resp.ConceptMeta).Error; err != nil {
		return nil, err
	}
	if err := db.Order("form_id ASC").Find(&resp.FormMeta).Error; err != nil {
		return nil, err
	}
	if err := db.Order("form_id ASC, id ASC").Find(&resp.FormSynonyms).Error; err != nil {
		return nil, err
	}
	if err := db.Where("is_active = TRUE").Order("direction_id ASC").Find(&resp.Directions).Error; err != nil {
		return nil, err
	}

	return resp, nil
}

type DirectionListQuery struct {
	Page           int
	PageSize       int
	Search         string
	DirectionCode  string
	SourceLangCode string
	TargetLangCode string
	CategoryID     uint64
	Unassigned     string
	Active         string
	SortBy         string
	SortDir        string
}

type DirectionPagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasPrev    bool  `json:"has_prev"`
	HasNext    bool  `json:"has_next"`
}

type DirectionSort struct {
	SortBy  string `json:"sort_by"`
	SortDir string `json:"sort_dir"`
}

type DirectionListResponse struct {
	Items      []TrainingDirectionView `json:"items"`
	Total      int64                   `json:"total"`
	Pagination DirectionPagination     `json:"pagination"`
	Sort       DirectionSort           `json:"sort"`
}

var directionListSelectColumns = []string{
	"direction_id",
	"concept_id",
	"direction_code",
	"source_lang_code",
	"target_lang_code",
	"source_form_id",
	"source_value",
	"target_form_id",
	"target_value",
	"category_id",
	"category_slug",
	"category_name_ru",
	"category_name_en",
	"category_name_de",
	"cefr_level",
	"importance_score",
	"concept_base_difficulty",
	"source_form_score",
	"target_form_score",
	"direction_bias",
	"synonym_relief",
	"final_difficulty",
	"is_active",
	"created_at",
}

func ListDirections(db *gorm.DB, params DirectionListQuery) (*DirectionListResponse, error) {
	page := normalizePage(params.Page)
	pageSize := normalizePageSize(params.PageSize)
	sortBy := normalizeDirectionSortBy(params.SortBy)
	sortDir := normalizeDirectionSortDir(params.SortDir)

	baseQuery := db.Model(&TrainingDirectionView{})
	baseQuery = applyDirectionFilters(baseQuery, params)

	var total int64
	if err := baseQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	items := make([]TrainingDirectionView, 0, pageSize)
	offset := (page - 1) * pageSize
	orderExpr := directionSortColumn(sortBy) + " " + sortDir + ", direction_id ASC"
	if err := baseQuery.Session(&gorm.Session{}).Select(directionListSelectColumns).Order(orderExpr).Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return &DirectionListResponse{
		Items: items,
		Total: total,
		Pagination: DirectionPagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
			HasPrev:    page > 1,
			HasNext:    totalPages > 0 && page < totalPages,
		},
		Sort: DirectionSort{
			SortBy:  sortBy,
			SortDir: sortDir,
		},
	}, nil
}

func applyDirectionFilters(query *gorm.DB, params DirectionListQuery) *gorm.DB {
	active := strings.ToLower(strings.TrimSpace(params.Active))
	switch active {
	case "all":
	case "false", "0", "no":
		query = query.Where("is_active = FALSE")
	default:
		query = query.Where("is_active = TRUE")
	}

	if directionCode := strings.TrimSpace(params.DirectionCode); directionCode != "" && directionCode != "all" {
		query = query.Where("direction_code = ?", directionCode)
	}
	if sourceLang := strings.TrimSpace(params.SourceLangCode); sourceLang != "" && sourceLang != "all" {
		query = query.Where("source_lang_code = ?", sourceLang)
	}
	if targetLang := strings.TrimSpace(params.TargetLangCode); targetLang != "" && targetLang != "all" {
		query = query.Where("target_lang_code = ?", targetLang)
	}
	if params.CategoryID > 0 {
		query = query.Where("category_id = ?", params.CategoryID)
	} else if isTruthyFilter(params.Unassigned) {
		query = query.Where("category_id IS NULL")
	}
	if search := strings.TrimSpace(params.Search); search != "" {
		like := "%" + search + "%"
		query = query.Where(`
			(
				direction_code ILIKE ?
				OR source_value ILIKE ?
				OR target_value ILIKE ?
				OR source_lang_code ILIKE ?
				OR target_lang_code ILIKE ?
				OR category_slug ILIKE ?
				OR category_name_ru ILIKE ?
				OR category_name_en ILIKE ?
				OR category_name_de ILIKE ?
				OR cefr_level ILIKE ?
			)
		`, like, like, like, like, like, like, like, like, like, like)
	}

	return query
}

func isTruthyFilter(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func normalizePage(value int) int {
	if value < 1 {
		return 1
	}
	return value
}

func normalizePageSize(value int) int {
	switch {
	case value <= 0:
		return 100
	case value > 250:
		return 250
	default:
		return value
	}
}

func normalizeDirectionSortBy(value string) string {
	sortBy := strings.ToLower(strings.TrimSpace(value))
	if _, ok := allowedDirectionSortColumns[sortBy]; ok {
		return sortBy
	}
	return "direction_id"
}

func normalizeDirectionSortDir(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "desc":
		return "desc"
	default:
		return "asc"
	}
}

func directionSortColumn(sortBy string) string {
	if column, ok := allowedDirectionSortColumns[sortBy]; ok {
		return column
	}
	return "direction_id"
}

var allowedDirectionSortColumns = map[string]string{
	"direction_id":     "direction_id",
	"direction_code":   "direction_code",
	"source_value":     "source_value",
	"target_value":     "target_value",
	"source_lang_code": "source_lang_code",
	"target_lang_code": "target_lang_code",
	"category_name_ru": "category_name_ru",
	"category_name_en": "category_name_en",
	"category_name_de": "category_name_de",
	"cefr_level":       "cefr_level",
	"final_difficulty": "final_difficulty",
	"is_active":        "is_active",
	"created_at":       "created_at",
}
