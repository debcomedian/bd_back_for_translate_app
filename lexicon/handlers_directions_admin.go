package lexicon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type updateDirectionRequest struct {
	CategoryID            *uint64  `json:"category_id"`
	CefrLevel             *string  `json:"cefr_level"`
	SourceLangCode        *string  `json:"source_lang_code"`
	TargetLangCode        *string  `json:"target_lang_code"`
	SourceValue           *string  `json:"source_value"`
	TargetValue           *string  `json:"target_value"`
	ImportanceScore       *int     `json:"importance_score"`
	ConceptBaseDifficulty *float64 `json:"concept_base_difficulty"`
	SourceFormScore       *float64 `json:"source_form_score"`
	TargetFormScore       *float64 `json:"target_form_score"`
	DirectionBias         *float64 `json:"direction_bias"`
	SynonymRelief         *float64 `json:"synonym_relief"`
	FinalDifficulty       *float64 `json:"final_difficulty"`
	IsActive              *bool    `json:"is_active"`
}

type updateCategoryDirectionsRequest struct {
	// DirectionIDs are representative directions selected by administrator.
	// The server stores them in category_direction_assignments and applies
	// the category to their lexical concepts, so neighbour directions are
	// included in the category automatically.
	DirectionIDs []uint64 `json:"direction_ids"`
	Mode         string   `json:"mode"`
}

type categoryDirectionSelectionResponse struct {
	CategoryID                 uint64                  `json:"category_id"`
	Items                      []TrainingDirectionView `json:"items"`
	DirectionIDs               []uint64                `json:"direction_ids"`
	ConceptIDs                 []uint64                `json:"concept_ids"`
	RepresentativeCount        int                     `json:"representative_count"`
	AssociatedDirectionCount   int64                   `json:"associated_direction_count"`
	HasExplicitRepresentatives bool                    `json:"has_explicit_representatives"`
}

type clientError string

func (e clientError) Error() string { return string(e) }

const maxCategoryDirectionSelection = 5000

func (h *Handler) UpdateDirection(c *gin.Context) {
	directionID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req updateDirectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "Некорректное тело JSON-запроса")
		return
	}

	if err := validateDirectionUpdateRequest(req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	var result TrainingDirectionView
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var direction TrainingDirection
		if err := tx.Where("id = ?", directionID).First(&direction).Error; err != nil {
			return err
		}

		sourceLang := direction.SourceLangCode
		targetLang := direction.TargetLangCode
		if req.SourceLangCode != nil {
			sourceLang = NormalizeValue(*req.SourceLangCode)
		}
		if req.TargetLangCode != nil {
			targetLang = NormalizeValue(*req.TargetLangCode)
		}
		if sourceLang == "" || targetLang == "" {
			return fmt.Errorf("исходный и целевой язык должны быть выбраны")
		}
		if sourceLang == targetLang {
			return fmt.Errorf("исходный и целевой язык не должны совпадать")
		}
		if err := ensureLanguageExists(tx, sourceLang); err != nil {
			return err
		}
		if err := ensureLanguageExists(tx, targetLang); err != nil {
			return err
		}

		sourceFormID := direction.SourceFormID
		targetFormID := direction.TargetFormID
		if req.SourceLangCode != nil || req.TargetLangCode != nil {
			sourceForm, err := findOrCreateConceptForm(tx, direction.ConceptID, sourceLang, req.SourceValue)
			if err != nil {
				return err
			}
			targetForm, err := findOrCreateConceptForm(tx, direction.ConceptID, targetLang, req.TargetValue)
			if err != nil {
				return err
			}
			if sourceForm.ID == targetForm.ID {
				return fmt.Errorf("исходная и целевая форма не должны совпадать")
			}
			if err := ensureDirectionPairAvailable(tx, direction.ConceptID, sourceForm.ID, targetForm.ID, directionID); err != nil {
				return err
			}
			sourceFormID = sourceForm.ID
			targetFormID = targetForm.ID
		}

		directionUpdates := map[string]any{}
		if req.SourceLangCode != nil || req.TargetLangCode != nil {
			directionUpdates["source_form_id"] = sourceFormID
			directionUpdates["target_form_id"] = targetFormID
			directionUpdates["source_lang_code"] = sourceLang
			directionUpdates["target_lang_code"] = targetLang
			directionUpdates["direction_code"] = sourceLang + "_" + targetLang
		}
		if req.IsActive != nil {
			directionUpdates["is_active"] = *req.IsActive
		}
		if len(directionUpdates) > 0 {
			if err := tx.Model(&TrainingDirection{}).
				Where("id = ?", directionID).
				Updates(directionUpdates).Error; err != nil {
				return err
			}
		}

		if err := updateDirectionForm(tx, sourceFormID, req.SourceValue); err != nil {
			return err
		}
		if err := updateDirectionForm(tx, targetFormID, req.TargetValue); err != nil {
			return err
		}

		if req.CategoryID != nil {
			if *req.CategoryID == 0 {
				if err := tx.Model(&LexicalConcept{}).
					Where("id = ?", direction.ConceptID).
					Update("category_id", nil).Error; err != nil {
					return err
				}
			} else {
				if err := ensureCategoryExists(tx, *req.CategoryID); err != nil {
					return err
				}
				if err := tx.Model(&LexicalConcept{}).
					Where("id = ?", direction.ConceptID).
					Update("category_id", *req.CategoryID).Error; err != nil {
					return err
				}
			}
		}

		conceptMetaUpdates := map[string]any{}
		if req.CefrLevel != nil {
			cefr := normalizeCEFR(*req.CefrLevel)
			if cefr == "" {
				conceptMetaUpdates["cefr_level"] = nil
			} else {
				conceptMetaUpdates["cefr_level"] = cefr
			}
		}
		if req.ImportanceScore != nil {
			conceptMetaUpdates["importance_score"] = *req.ImportanceScore
		}
		if req.ConceptBaseDifficulty != nil {
			conceptMetaUpdates["base_difficulty"] = round3(*req.ConceptBaseDifficulty)
		}
		if len(conceptMetaUpdates) > 0 {
			conceptMetaUpdates["calculated_at"] = time.Now()
			if err := tx.Model(&LexicalConceptMeta{}).
				Where("concept_id = ?", direction.ConceptID).
				Updates(conceptMetaUpdates).Error; err != nil {
				return err
			}
		}

		metaUpdates := map[string]any{}
		if req.SourceFormScore != nil {
			metaUpdates["source_form_score"] = round3(*req.SourceFormScore)
		}
		if req.TargetFormScore != nil {
			metaUpdates["target_form_score"] = round3(*req.TargetFormScore)
		}
		if req.ConceptBaseDifficulty != nil {
			metaUpdates["concept_base_difficulty"] = round3(*req.ConceptBaseDifficulty)
		}
		if req.DirectionBias != nil {
			metaUpdates["direction_bias"] = round3(*req.DirectionBias)
		}
		if req.SynonymRelief != nil {
			metaUpdates["synonym_relief"] = round3(*req.SynonymRelief)
		}
		if req.FinalDifficulty != nil {
			metaUpdates["final_difficulty"] = round3(*req.FinalDifficulty)
		}
		if len(metaUpdates) > 0 {
			metaUpdates["meta_version"] = gorm.Expr("meta_version + 1")
			metaUpdates["calculated_at"] = time.Now()
			if err := tx.Model(&TrainingDirectionMeta{}).
				Where("direction_id = ?", directionID).
				Updates(metaUpdates).Error; err != nil {
				return err
			}
		}

		if err := createAuditLog(tx, getActorID(c), "update_direction", "training_direction", directionID, req); err != nil {
			return err
		}

		if _, err := publishSnapshotVersion(tx); err != nil {
			return err
		}

		return tx.Where("direction_id = ?", directionID).First(&result).Error
	})

	if err != nil {
		respondAdminMutationError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetCategoryDirections(c *gin.Context) {
	categoryID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := ensureCategoryExists(h.DB, categoryID); err != nil {
		respondGormError(c, err)
		return
	}

	resp, err := ListDirections(h.DB, DirectionListQuery{
		Page:           parseIntQuery(c, "page", 1),
		PageSize:       parseIntQuery(c, "page_size", 100),
		Search:         c.Query("q"),
		DirectionCode:  c.Query("direction_code"),
		SourceLangCode: c.Query("source_lang_code"),
		TargetLangCode: c.Query("target_lang_code"),
		CategoryID:     categoryID,
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

func (h *Handler) GetCategoryDirectionSelection(c *gin.Context) {
	categoryID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := ensureCategoryExists(h.DB, categoryID); err != nil {
		respondGormError(c, err)
		return
	}
	if err := ensureCategoryAssignmentTable(h.DB); err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	selectedSortByRaw := strings.TrimSpace(c.Query("sort_by"))
	selectedSortBy := normalizeDirectionSortBy(selectedSortByRaw)
	selectedSortDir := normalizeDirectionSortDir(c.Query("sort_dir"))
	items, explicit, err := loadCategoryRepresentativeDirections(h.DB, categoryID, selectedSortBy, selectedSortDir, selectedSortByRaw != "")
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	var associatedDirections int64
	if err := h.DB.Model(&TrainingDirectionView{}).
		Where("category_id = ?", categoryID).
		Count(&associatedDirections).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	directionIDs := make([]uint64, 0, len(items))
	conceptIDs := make([]uint64, 0, len(items))
	seenConcepts := make(map[uint64]struct{}, len(items))
	for _, item := range items {
		directionIDs = append(directionIDs, item.DirectionID)
		if _, ok := seenConcepts[item.ConceptID]; !ok {
			seenConcepts[item.ConceptID] = struct{}{}
			conceptIDs = append(conceptIDs, item.ConceptID)
		}
	}

	c.JSON(http.StatusOK, categoryDirectionSelectionResponse{
		CategoryID:                 categoryID,
		Items:                      items,
		DirectionIDs:               directionIDs,
		ConceptIDs:                 conceptIDs,
		RepresentativeCount:        len(items),
		AssociatedDirectionCount:   associatedDirections,
		HasExplicitRepresentatives: explicit,
	})
}

func (h *Handler) UpdateCategoryDirections(c *gin.Context) {
	categoryID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}

	var req updateCategoryDirectionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", "Некорректное тело JSON-запроса")
		return
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "replace"
	}
	if mode != "replace" && mode != "add" && mode != "remove" {
		respondError(c, http.StatusBadRequest, "bad_request", "Недопустимый режим обновления категории")
		return
	}

	ids := uniqueUint64(req.DirectionIDs)
	if len(ids) > maxCategoryDirectionSelection {
		respondError(c, http.StatusBadRequest, "bad_request", fmt.Sprintf("Нельзя сохранить больше %d направлений за один запрос", maxCategoryDirectionSelection))
		return
	}
	if mode != "replace" && len(ids) == 0 {
		respondError(c, http.StatusBadRequest, "bad_request", "Список направлений пуст")
		return
	}

	var result struct {
		CategoryID               uint64 `json:"category_id"`
		Mode                     string `json:"mode"`
		DirectionCount           int    `json:"direction_count"`
		RepresentativeCount      int    `json:"representative_count"`
		ConceptCount             int    `json:"concept_count"`
		AffectedConcepts         int64  `json:"affected_concepts"`
		AffectedDirections       int64  `json:"affected_directions"`
		AssociatedDirectionCount int64  `json:"associated_direction_count"`
		SnapshotVersion          int64  `json:"snapshot_version"`
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureCategoryExists(tx, categoryID); err != nil {
			return err
		}
		if err := ensureCategoryAssignmentTable(tx); err != nil {
			return err
		}

		representatives, err := loadCategoryRepresentativesByDirectionIDsStrict(tx, ids)
		if err != nil {
			return err
		}
		conceptIDs := conceptIDsFromRepresentatives(representatives)
		if mode == "replace" || mode == "add" {
			if err := ensureConceptsAssignableToCategory(tx, categoryID, conceptIDs); err != nil {
				return err
			}
		}

		var affected int64
		switch mode {
		case "replace":
			if err := tx.Where("category_id = ?", categoryID).Delete(&CategoryDirectionAssignment{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&LexicalConcept{}).
				Where("category_id = ?", categoryID).
				Update("category_id", nil).Error; err != nil {
				return err
			}
			if len(conceptIDs) > 0 {
				res := tx.Model(&LexicalConcept{}).
					Where("id IN ?", conceptIDs).
					Update("category_id", categoryID)
				if res.Error != nil {
					return res.Error
				}
				affected = res.RowsAffected
			}
			if err := insertCategoryRepresentatives(tx, categoryID, representatives); err != nil {
				return err
			}
		case "add":
			if len(conceptIDs) > 0 {
				res := tx.Model(&LexicalConcept{}).
					Where("id IN ?", conceptIDs).
					Update("category_id", categoryID)
				if res.Error != nil {
					return res.Error
				}
				affected = res.RowsAffected
			}
			if err := upsertCategoryRepresentatives(tx, categoryID, representatives); err != nil {
				return err
			}
		case "remove":
			if len(conceptIDs) > 0 {
				if err := tx.Where("category_id = ? AND concept_id IN ?", categoryID, conceptIDs).
					Delete(&CategoryDirectionAssignment{}).Error; err != nil {
					return err
				}
				res := tx.Model(&LexicalConcept{}).
					Where("id IN ? AND category_id = ?", conceptIDs, categoryID).
					Update("category_id", nil)
				if res.Error != nil {
					return res.Error
				}
				affected = res.RowsAffected
			}
		}

		var affectedDirections int64
		if err := tx.Model(&TrainingDirectionView{}).
			Where("category_id = ?", categoryID).
			Count(&affectedDirections).Error; err != nil {
			return err
		}

		if err := createAuditLog(tx, getActorID(c), "update_category_directions", "category", categoryID, req); err != nil {
			return err
		}

		snapshot, err := publishSnapshotVersion(tx)
		if err != nil {
			return err
		}

		result.CategoryID = categoryID
		result.Mode = mode
		result.DirectionCount = len(ids)
		result.RepresentativeCount = len(representatives)
		result.ConceptCount = len(conceptIDs)
		result.AffectedConcepts = affected
		result.AffectedDirections = affectedDirections
		result.AssociatedDirectionCount = affectedDirections
		if snapshot != nil {
			result.SnapshotVersion = snapshot.VersionCode
		}
		return nil
	})

	if err != nil {
		respondAdminMutationError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

type categoryDirectionRepresentative struct {
	DirectionID uint64
	ConceptID   uint64
}

func ensureCategoryAssignmentTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS lexicon.category_direction_assignments (
    category_id  BIGINT NOT NULL REFERENCES lexicon.categories(id) ON DELETE CASCADE,
    direction_id BIGINT NOT NULL REFERENCES lexicon.training_directions(id) ON DELETE CASCADE,
    concept_id   BIGINT NOT NULL REFERENCES lexicon.lexical_concepts(id) ON DELETE CASCADE,
    position     INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (category_id, direction_id),
    CONSTRAINT chk_category_direction_assignments_position CHECK (position >= 0)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_category_direction_assignments_category_concept
    ON lexicon.category_direction_assignments(category_id, concept_id);
CREATE INDEX IF NOT EXISTS idx_category_direction_assignments_category_position
    ON lexicon.category_direction_assignments(category_id, position, direction_id);
CREATE INDEX IF NOT EXISTS idx_category_direction_assignments_direction
    ON lexicon.category_direction_assignments(direction_id);
`).Error
}

func loadCategoryRepresentativeDirections(db *gorm.DB, categoryID uint64, sortBy string, sortDir string, useSort bool) ([]TrainingDirectionView, bool, error) {
	items := make([]TrainingDirectionView, 0)
	orderExpr := "a.position ASC, a.direction_id ASC"
	if useSort {
		orderExpr = "v." + directionSortColumn(sortBy) + " " + sortDir + ", v.direction_id ASC"
	}
	err := db.Table("lexicon.category_direction_assignments AS a").
		Select(prefixedDirectionListSelect("v")).
		Joins("JOIN lexicon.training_direction_view AS v ON v.direction_id = a.direction_id").
		Where("a.category_id = ?", categoryID).
		Order(orderExpr).
		Find(&items).Error
	if err != nil {
		return nil, false, err
	}
	if len(items) > 0 {
		return items, true, nil
	}

	// Backward-compatible fallback for databases that already had category_id
	// on concepts before category_direction_assignments was introduced.
	fallbackOrder := "concept_id ASC, direction_id ASC"
	if useSort {
		fallbackOrder = "concept_id ASC, " + directionSortColumn(sortBy) + " " + sortDir + ", direction_id ASC"
	}
	fallbackSQL := `
SELECT DISTINCT ON (concept_id)
    direction_id,
    concept_id,
    direction_code,
    source_lang_code,
    target_lang_code,
    source_form_id,
    source_value,
    target_form_id,
    target_value,
    category_id,
    category_slug,
    category_name_ru,
    category_name_en,
    category_name_de,
    cefr_level,
    importance_score,
    concept_base_difficulty,
    source_form_score,
    target_form_score,
    direction_bias,
    synonym_relief,
    final_difficulty,
    is_active,
    created_at
FROM lexicon.training_direction_view
WHERE category_id = ?
ORDER BY ` + fallbackOrder
	fallback := make([]TrainingDirectionView, 0)
	if err := db.Raw(fallbackSQL, categoryID).Scan(&fallback).Error; err != nil {
		return nil, false, err
	}
	return fallback, false, nil
}

func prefixedDirectionListSelect(alias string) string {
	columns := make([]string, 0, len(directionListSelectColumns))
	for _, column := range directionListSelectColumns {
		columns = append(columns, alias+"."+column)
	}
	return strings.Join(columns, ", ")
}

func loadCategoryRepresentativesByDirectionIDsStrict(db *gorm.DB, directionIDs []uint64) ([]categoryDirectionRepresentative, error) {
	if len(directionIDs) == 0 {
		return []categoryDirectionRepresentative{}, nil
	}

	type directionConcept struct {
		ID        uint64
		ConceptID uint64
	}
	var rows []directionConcept
	if err := db.Model(&TrainingDirection{}).
		Select("id, concept_id").
		Where("id IN ?", directionIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	found := make(map[uint64]uint64, len(rows))
	for _, row := range rows {
		found[row.ID] = row.ConceptID
	}
	if len(found) != len(directionIDs) {
		missing := make([]string, 0)
		for _, id := range directionIDs {
			if _, ok := found[id]; !ok {
				missing = append(missing, fmt.Sprintf("%d", id))
			}
		}
		return nil, clientError("не найдены направления: " + strings.Join(missing, ", "))
	}

	// One explicit representative per concept. If the request contains several
	// directions of one concept, the later direction wins and becomes the visible
	// representative in the selected block.
	result := make([]categoryDirectionRepresentative, 0, len(directionIDs))
	indexByConcept := make(map[uint64]int, len(directionIDs))
	for _, id := range directionIDs {
		conceptID := found[id]
		if idx, ok := indexByConcept[conceptID]; ok {
			result[idx].DirectionID = id
			continue
		}
		indexByConcept[conceptID] = len(result)
		result = append(result, categoryDirectionRepresentative{DirectionID: id, ConceptID: conceptID})
	}
	return result, nil
}

func conceptIDsFromRepresentatives(reps []categoryDirectionRepresentative) []uint64 {
	ids := make([]uint64, 0, len(reps))
	for _, rep := range reps {
		ids = append(ids, rep.ConceptID)
	}
	return ids
}

func ensureConceptsAssignableToCategory(db *gorm.DB, categoryID uint64, conceptIDs []uint64) error {
	if len(conceptIDs) == 0 {
		return nil
	}
	type occupiedConcept struct {
		ID         uint64
		CategoryID uint64
	}
	var rows []occupiedConcept
	if err := db.Table("lexicon.lexical_concepts").
		Select("id, category_id").
		Where("id IN ? AND category_id IS NOT NULL AND category_id <> ?", conceptIDs, categoryID).
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, fmt.Sprintf("карточка %d уже находится в категории %d", row.ID, row.CategoryID))
	}
	return clientError("часть выбранных карточек уже закреплена за другой категорией: " + strings.Join(parts, "; "))
}

func insertCategoryRepresentatives(tx *gorm.DB, categoryID uint64, reps []categoryDirectionRepresentative) error {
	if len(reps) == 0 {
		return nil
	}
	rows := make([]CategoryDirectionAssignment, 0, len(reps))
	for index, rep := range reps {
		rows = append(rows, CategoryDirectionAssignment{
			CategoryID:  categoryID,
			DirectionID: rep.DirectionID,
			ConceptID:   rep.ConceptID,
			Position:    index + 1,
			CreatedAt:   time.Now(),
		})
	}
	return tx.Create(&rows).Error
}

func upsertCategoryRepresentatives(tx *gorm.DB, categoryID uint64, reps []categoryDirectionRepresentative) error {
	if len(reps) == 0 {
		return nil
	}
	for _, rep := range reps {
		if err := tx.Where("category_id = ? AND concept_id = ?", categoryID, rep.ConceptID).
			Delete(&CategoryDirectionAssignment{}).Error; err != nil {
			return err
		}
	}

	var maxPosition int
	if err := tx.Model(&CategoryDirectionAssignment{}).
		Where("category_id = ?", categoryID).
		Select("COALESCE(MAX(position), 0)").
		Scan(&maxPosition).Error; err != nil {
		return err
	}

	rows := make([]CategoryDirectionAssignment, 0, len(reps))
	for index, rep := range reps {
		rows = append(rows, CategoryDirectionAssignment{
			CategoryID:  categoryID,
			DirectionID: rep.DirectionID,
			ConceptID:   rep.ConceptID,
			Position:    maxPosition + index + 1,
			CreatedAt:   time.Now(),
		})
	}
	return tx.Create(&rows).Error
}

func validateDirectionUpdateRequest(req updateDirectionRequest) error {
	if req.CategoryID == nil && req.CefrLevel == nil && req.SourceLangCode == nil && req.TargetLangCode == nil && req.SourceValue == nil && req.TargetValue == nil && req.ImportanceScore == nil && req.ConceptBaseDifficulty == nil && req.SourceFormScore == nil && req.TargetFormScore == nil && req.DirectionBias == nil && req.SynonymRelief == nil && req.FinalDifficulty == nil && req.IsActive == nil {
		return fmt.Errorf("не передано ни одно поле для изменения")
	}
	if req.CefrLevel != nil {
		cefr := normalizeCEFR(*req.CefrLevel)
		if cefr != "" && !isAllowedCEFR(cefr) {
			return fmt.Errorf("недопустимый уровень")
		}
	}
	if req.ImportanceScore != nil && *req.ImportanceScore < 0 {
		return fmt.Errorf("важность не может быть отрицательной")
	}
	for label, value := range map[string]*float64{
		"базовая сложность карточки": req.ConceptBaseDifficulty,
		"оценка исходной формы":      req.SourceFormScore,
		"оценка целевой формы":       req.TargetFormScore,
		"снижение за синонимы":       req.SynonymRelief,
		"итоговая сложность":         req.FinalDifficulty,
	} {
		if value != nil && (*value < 0 || *value > 10) {
			return fmt.Errorf("%s должна быть в диапазоне от 0 до 10", label)
		}
	}
	if req.DirectionBias != nil && (*req.DirectionBias < -10 || *req.DirectionBias > 10) {
		return fmt.Errorf("поправка направления должна быть в диапазоне от -10 до 10")
	}
	for label, value := range map[string]*string{
		"исходная форма": req.SourceValue,
		"целевая форма":  req.TargetValue,
	} {
		if value != nil && strings.TrimSpace(*value) == "" {
			return fmt.Errorf("поле %q не должно быть пустым", label)
		}
	}
	return nil
}

func findOrCreateConceptForm(tx *gorm.DB, conceptID uint64, langCode string, value *string) (*LexicalForm, error) {
	var form LexicalForm
	err := tx.Where("concept_id = ? AND lang_code = ? AND is_primary = TRUE", conceptID, langCode).First(&form).Error
	if err == nil {
		return &form, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("для языка %s у карточки нет формы", langCode)
	}
	prepared := normalizeLexeme(*value)
	if prepared == "" {
		return nil, fmt.Errorf("форма для языка %s не должна быть пустой", langCode)
	}
	form = LexicalForm{
		ConceptID:       conceptID,
		LangCode:        langCode,
		Value:           prepared,
		NormalizedValue: NormalizeValue(prepared),
		IsPrimary:       true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := tx.Create(&form).Error; err != nil {
		return nil, err
	}
	return &form, nil
}

func ensureDirectionPairAvailable(tx *gorm.DB, conceptID, sourceFormID, targetFormID, currentDirectionID uint64) error {
	var count int64
	if err := tx.Model(&TrainingDirection{}).
		Where("concept_id = ? AND source_form_id = ? AND target_form_id = ? AND id <> ?", conceptID, sourceFormID, targetFormID, currentDirectionID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return clientError("такое направление для этой карточки уже существует")
	}
	return nil
}

func updateDirectionForm(tx *gorm.DB, formID uint64, value *string) error {
	if value == nil {
		return nil
	}
	prepared := normalizeLexeme(*value)
	if prepared == "" {
		return fmt.Errorf("форма не должна быть пустой")
	}
	return tx.Model(&LexicalForm{}).Where("id = ?", formID).Updates(map[string]any{
		"value":            prepared,
		"normalized_value": NormalizeValue(prepared),
		"updated_at":       time.Now(),
	}).Error
}

func normalizeCEFR(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func isAllowedCEFR(value string) bool {
	switch value {
	case "A1", "A2", "B1", "B2", "C1", "C2":
		return true
	default:
		return false
	}
}

func ensureCategoryExists(db *gorm.DB, categoryID uint64) error {
	if categoryID == 0 {
		return fmt.Errorf("категория не указана")
	}
	var count int64
	if err := db.Model(&Category{}).Where("id = ?", categoryID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func ensureLanguageExists(db *gorm.DB, code string) error {
	code = NormalizeValue(code)
	if code == "" {
		return fmt.Errorf("код языка не указан")
	}
	var count int64
	if err := db.Model(&Language{}).Where("code = ? AND is_active = TRUE", code).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("язык %q не найден или отключён", code)
	}
	return nil
}

func loadConceptIDsByDirectionIDsStrict(db *gorm.DB, directionIDs []uint64) ([]uint64, error) {
	if len(directionIDs) == 0 {
		return []uint64{}, nil
	}

	type directionConcept struct {
		ID        uint64
		ConceptID uint64
	}
	var rows []directionConcept
	if err := db.Model(&TrainingDirection{}).
		Select("id, concept_id").
		Where("id IN ?", directionIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) != len(directionIDs) {
		found := make(map[uint64]struct{}, len(rows))
		for _, row := range rows {
			found[row.ID] = struct{}{}
		}
		missing := make([]string, 0)
		for _, id := range directionIDs {
			if _, ok := found[id]; !ok {
				missing = append(missing, fmt.Sprintf("%d", id))
			}
		}
		return nil, clientError("не найдены направления: " + strings.Join(missing, ", "))
	}

	seen := make(map[uint64]struct{}, len(rows))
	conceptIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.ConceptID]; ok {
			continue
		}
		seen[row.ConceptID] = struct{}{}
		conceptIDs = append(conceptIDs, row.ConceptID)
	}
	return conceptIDs, nil
}

func uniqueUint64(values []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(values))
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func getActorID(c *gin.Context) uint64 {
	value, ok := c.Get("uid")
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case int:
		return uint64(v)
	case uint:
		return uint64(v)
	case uint64:
		return v
	case float64:
		return uint64(v)
	default:
		return 0
	}
}

func createAuditLog(tx *gorm.DB, actorID uint64, actionType, entityType string, entityID uint64, payload any) error {
	if actorID == 0 {
		return nil
	}

	var adminCount int64
	if err := tx.Table("public.admin_users").Where("id = ?", actorID).Count(&adminCount).Error; err != nil {
		return nil
	}
	if adminCount == 0 {
		return nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return tx.Create(&AuditLog{
		AdminUserID: actorID,
		ActionType:  actionType,
		EntityType:  entityType,
		EntityID:    &entityID,
		PayloadJSON: data,
	}).Error
}

func respondAdminMutationError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if publicErr, ok := err.(clientError); ok {
		respondError(c, http.StatusBadRequest, "bad_request", publicErr.Error())
		return
	}
	if err == gorm.ErrRecordNotFound {
		respondError(c, http.StatusNotFound, "not_found", "Запись не найдена")
		return
	}
	respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
}

func respondGormError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if err == gorm.ErrRecordNotFound {
		respondError(c, http.StatusNotFound, "not_found", "Запись не найдена")
		return
	}
	respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
}
