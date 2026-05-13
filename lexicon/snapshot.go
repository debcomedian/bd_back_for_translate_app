package lexicon

import (
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

type DirectionListResponse struct {
	Items []TrainingDirectionView `json:"items"`
	Total int                     `json:"total"`
}

func ListDirections(db *gorm.DB) (*DirectionListResponse, error) {
	var items []TrainingDirectionView
	if err := db.Where("is_active = TRUE").Order("direction_id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return &DirectionListResponse{Items: items, Total: len(items)}, nil
}
