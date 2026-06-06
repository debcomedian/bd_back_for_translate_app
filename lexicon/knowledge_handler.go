package lexicon

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetKnowledgeSummary(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	var progress []UserDirectionProgress
	if err := h.DB.Where("user_id = ?", uid).Order("direction_id ASC").Find(&progress).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	directionIDs := make([]uint64, 0, len(progress))
	for _, item := range progress {
		directionIDs = append(directionIDs, item.DirectionID)
	}

	difficultyByDirection := map[uint64]float64{}
	if len(directionIDs) > 0 {
		var metas []TrainingDirectionMeta
		if err := h.DB.Where("direction_id IN ?", directionIDs).Find(&metas).Error; err != nil {
			respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		for _, meta := range metas {
			difficultyByDirection[meta.DirectionID] = meta.FinalDifficulty
		}
	}

	c.JSON(http.StatusOK, CalculateUserKnowledgeSummary(uid, progress, difficultyByDirection))
}
