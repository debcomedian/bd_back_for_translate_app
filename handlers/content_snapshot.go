package handlers

import (
	"bd_back_for_translate_app/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type snapshotResponse struct {
	SnapshotVersion *models.ContentSnapshotVersion `json:"snapshot_version,omitempty"`
	Categories      []models.Category              `json:"categories"`
	Words           []models.Word                  `json:"words"`
	WordsMetaBase   []models.WordMetaBase          `json:"words_meta_base"`
	WordSynonyms    []models.WordSynonym           `json:"word_synonyms"`
}

func GetContentSnapshot(c *gin.Context) {
	var snapshot models.ContentSnapshotVersion
	err := DB.Where("snapshot_type = ? AND is_active = TRUE", "words_base").Order("version_code DESC").First(&snapshot).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		handleDBErr(c, err)
		return
	}

	var categories []models.Category
	if err := DB.Order("id ASC").Find(&categories).Error; err != nil {
		handleDBErr(c, err)
		return
	}

	var words []models.Word
	if err := DB.Where("is_active = TRUE").Order("id ASC").Find(&words).Error; err != nil {
		handleDBErr(c, err)
		return
	}

	var metas []models.WordMetaBase
	if err := DB.Order("word_id ASC").Find(&metas).Error; err != nil {
		handleDBErr(c, err)
		return
	}

	var synonyms []models.WordSynonym
	if err := DB.Order("id ASC").Find(&synonyms).Error; err != nil {
		handleDBErr(c, err)
		return
	}

	resp := snapshotResponse{
		Categories:    categories,
		Words:         words,
		WordsMetaBase: metas,
		WordSynonyms:  synonyms,
	}
	if snapshot.ID != 0 {
		resp.SnapshotVersion = &snapshot
	}
	respondOK(c, resp)
}
