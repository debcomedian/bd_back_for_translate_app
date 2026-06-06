package lexicon

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type syncPushRequest struct {
	Events []syncEventInput `json:"events"`
}
type syncEventInput struct {
	EventID         string          `json:"event_id"`
	DeviceID        string          `json:"device_id"`
	EntityType      string          `json:"entity_type"`
	EntityID        *uint64         `json:"entity_id"`
	EventType       string          `json:"event_type"`
	Payload         json.RawMessage `json:"payload"`
	ClientCreatedAt time.Time       `json:"client_created_at"`
}
type attemptPayload struct {
	DirectionID    uint64     `json:"direction_id"`
	Result         string     `json:"result"`
	ResponseValue  *string    `json:"response_value"`
	ResponseTimeMS int        `json:"response_time_ms"`
	AttemptedAt    *time.Time `json:"attempted_at"`
}
type progressPayload struct {
	DirectionID             uint64     `json:"direction_id"`
	Box                     int        `json:"box"`
	RepeatCount             int        `json:"repeat_count"`
	CorrectCount            int        `json:"correct_count"`
	IncorrectCount          int        `json:"incorrect_count"`
	MasteryScore            float64    `json:"mastery_score"`
	HalfLifeDays            float64    `json:"half_life_days"`
	RecallProbability       float64    `json:"recall_probability"`
	LastResult              string     `json:"last_result"`
	LastResponseTimeMS      int        `json:"last_response_time_ms"`
	DifficultyAtLastAttempt float64    `json:"difficulty_at_last_attempt"`
	LastSeenAt              *time.Time `json:"last_seen_at"`
	NextDue                 *time.Time `json:"next_due"`
	UpdatedAt               *time.Time `json:"updated_at"`
}
type syncPushResponse struct {
	Accepted   int `json:"accepted"`
	Duplicates int `json:"duplicates"`
	Failed     int `json:"failed"`
}

func (h *Handler) SyncPush(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var req syncPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	resp := syncPushResponse{}
	for _, event := range req.Events {
		accepted, duplicate, err := h.processSyncEvent(uid, event)
		if duplicate {
			resp.Duplicates++
			continue
		}
		if err != nil {
			resp.Failed++
			continue
		}
		if accepted {
			resp.Accepted++
		}
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) processSyncEvent(uid uint64, event syncEventInput) (bool, bool, error) {
	if event.EventID == "" {
		return false, false, nil
	}
	var existing SyncEvent
	err := h.DB.Where("event_id = ?", event.EventID).First(&existing).Error
	if err == nil {
		return false, true, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return false, false, err
	}
	now := time.Now()
	row := SyncEvent{EventID: event.EventID, UserID: uid, DeviceID: event.DeviceID, EntityType: event.EntityType, EntityID: event.EntityID, EventType: event.EventType, PayloadJSON: []byte(event.Payload), ClientCreatedAt: event.ClientCreatedAt, ServerReceivedAt: now, Status: "received"}
	if row.ClientCreatedAt.IsZero() {
		row.ClientCreatedAt = now
	}
	if err := h.DB.Create(&row).Error; err != nil {
		return false, false, err
	}
	switch event.EventType {
	case "attempt":
		if err := h.applyAttemptEvent(uid, event); err != nil {
			return false, false, err
		}
	case "progress_update":
		if err := h.applyProgressEvent(uid, event); err != nil {
			return false, false, err
		}
	}
	return true, false, nil
}

func (h *Handler) applyAttemptEvent(uid uint64, event syncEventInput) error {
	var payload attemptPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}
	if payload.DirectionID == 0 && event.EntityID != nil {
		payload.DirectionID = *event.EntityID
	}
	var direction TrainingDirection
	if err := h.DB.Where("id = ?", payload.DirectionID).First(&direction).Error; err != nil {
		return err
	}
	var source, target LexicalForm
	if err := h.DB.Where("id = ?", direction.SourceFormID).First(&source).Error; err != nil {
		return err
	}
	if err := h.DB.Where("id = ?", direction.TargetFormID).First(&target).Error; err != nil {
		return err
	}
	attemptedAt := time.Now()
	if payload.AttemptedAt != nil {
		attemptedAt = *payload.AttemptedAt
	}
	result := normalizeAttemptResult(payload.Result)
	attempt := Attempt{UserID: uid, DirectionID: direction.ID, ConceptID: direction.ConceptID, SourceFormID: direction.SourceFormID, TargetFormID: direction.TargetFormID, DirectionCode: direction.DirectionCode, PromptValue: source.Value, ExpectedValue: target.Value, ResponseValue: payload.ResponseValue, Result: result, ResponseTimeMS: payload.ResponseTimeMS, DeviceID: event.DeviceID, AttemptedAt: attemptedAt}
	if err := h.DB.Create(&attempt).Error; err != nil {
		return err
	}
	return h.bumpProgressFromAttempt(uid, direction, result, payload.ResponseTimeMS, attemptedAt)
}

func (h *Handler) bumpProgressFromAttempt(uid uint64, direction TrainingDirection, result string, responseTimeMS int, at time.Time) error {
	var progress UserDirectionProgress
	err := h.DB.Where("user_id = ? AND direction_id = ?", uid, direction.ID).First(&progress).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		progress = UserDirectionProgress{UserID: uid, DirectionID: direction.ID, ConceptID: direction.ConceptID, DirectionCode: direction.DirectionCode}
	}

	result = normalizeAttemptResult(result)
	difficulty, err := h.getDirectionFinalDifficulty(direction.ID)
	if err != nil {
		return err
	}

	previousLastSeenAt := progress.LastSeenAt
	previousMastery := progress.MasteryScore
	progress.RepeatCount++
	if result == "correct" {
		progress.CorrectCount++
		progress.Box++
	} else if result == "partial" {
		progress.CorrectCount++
	} else {
		progress.IncorrectCount++
		if progress.Box > 0 {
			progress.Box--
		}
	}

	hlr := CalculateHalfLifeUpdate(HalfLifeInput{
		PreviousHalfLifeDays: progress.HalfLifeDays,
		PreviousMasteryScore: previousMastery,
		LastSeenAt:           previousLastSeenAt,
		AttemptedAt:          at,
		Result:               result,
		ResponseTimeMS:       responseTimeMS,
		Difficulty:           difficulty,
		RepeatCount:          progress.RepeatCount,
	})

	progress.MasteryScore = hlr.MasteryScore
	progress.HalfLifeDays = hlr.HalfLifeDays
	progress.RecallProbability = hlr.RecallProbability
	progress.LastResult = result
	progress.LastResponseTimeMS = responseTimeMS
	progress.DifficultyAtLastAttempt = hlr.Difficulty
	progress.LastSeenAt = &at
	progress.NextDue = &hlr.NextDue
	progress.UpdatedAt = time.Now()

	return h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "direction_id"}},
		DoUpdates: clause.AssignmentColumns(progressUpsertColumns()),
	}).Create(&progress).Error
}

func (h *Handler) getDirectionFinalDifficulty(directionID uint64) (float64, error) {
	var meta TrainingDirectionMeta
	if err := h.DB.Where("direction_id = ?", directionID).First(&meta).Error; err == nil {
		if meta.FinalDifficulty > 0 {
			return meta.FinalDifficulty, nil
		}
	} else if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	return defaultDifficultyForHLR, nil
}

func (h *Handler) applyProgressEvent(uid uint64, event syncEventInput) error {
	var payload progressPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}
	if payload.DirectionID == 0 && event.EntityID != nil {
		payload.DirectionID = *event.EntityID
	}
	var direction TrainingDirection
	if err := h.DB.Where("id = ?", payload.DirectionID).First(&direction).Error; err != nil {
		return err
	}
	updatedAt := time.Now()
	if payload.UpdatedAt != nil {
		updatedAt = *payload.UpdatedAt
	}
	progress := UserDirectionProgress{
		UserID:                  uid,
		DirectionID:             direction.ID,
		ConceptID:               direction.ConceptID,
		DirectionCode:           direction.DirectionCode,
		Box:                     payload.Box,
		RepeatCount:             payload.RepeatCount,
		CorrectCount:            payload.CorrectCount,
		IncorrectCount:          payload.IncorrectCount,
		MasteryScore:            payload.MasteryScore,
		HalfLifeDays:            payload.HalfLifeDays,
		RecallProbability:       payload.RecallProbability,
		LastResult:              payload.LastResult,
		LastResponseTimeMS:      payload.LastResponseTimeMS,
		DifficultyAtLastAttempt: payload.DifficultyAtLastAttempt,
		LastSeenAt:              payload.LastSeenAt,
		NextDue:                 payload.NextDue,
		UpdatedAt:               updatedAt,
	}
	return h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "direction_id"}},
		DoUpdates: clause.AssignmentColumns(progressUpsertColumns()),
	}).Create(&progress).Error
}

func progressUpsertColumns() []string {
	return []string{
		"concept_id",
		"direction_code",
		"box",
		"repeat_count",
		"correct_count",
		"incorrect_count",
		"mastery_score",
		"half_life_days",
		"recall_probability",
		"last_result",
		"last_response_time_ms",
		"difficulty_at_last_attempt",
		"last_seen_at",
		"next_due",
		"updated_at",
	}
}

func (h *Handler) SyncPull(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	clientVersion, _ := strconv.ParseInt(c.Query("snapshot_version"), 10, 64)
	var snapshot ContentSnapshotVersion
	err := h.DB.Where("snapshot_type = ? AND is_active = TRUE", directionalSnapshotType).Order("version_code DESC").First(&snapshot).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	var progress []UserDirectionProgress
	if err := h.DB.Where("user_id = ?", uid).Order("direction_id ASC").Find(&progress).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	needDownload := snapshot.ID != 0 && clientVersion != snapshot.VersionCode
	c.JSON(http.StatusOK, gin.H{"snapshot_version": snapshot, "need_download_snapshot": needDownload, "progress": progress})
}
