package handlers

import (
	"encoding/json"
	"time"

	"bd_back_for_translate_app/models"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type syncPushRequest struct {
	Events []syncPushEvent `json:"events" binding:"required"`
}

type syncPushEvent struct {
	EventID         string          `json:"event_id" binding:"required"`
	DeviceID        string          `json:"device_id" binding:"required"`
	EntityType      string          `json:"entity_type" binding:"required"`
	EntityID        *uint64         `json:"entity_id"`
	EventType       string          `json:"event_type" binding:"required"`
	Payload         json.RawMessage `json:"payload" binding:"required"`
	ClientCreatedAt time.Time       `json:"client_created_at" binding:"required"`
}

type syncPushResponse struct {
	Accepted   int `json:"accepted"`
	Duplicates int `json:"duplicates"`
	Failed     int `json:"failed"`
}

type attemptPayload struct {
	WordID         uint64     `json:"word_id"`
	Result         string     `json:"result"`
	ResponseValue  *string    `json:"response_value"`
	ResponseTimeMs int        `json:"response_time_ms"`
	AttemptedAt    *time.Time `json:"attempted_at"`
}

type progressPayload struct {
	WordID         uint64     `json:"word_id"`
	Box            int        `json:"box"`
	RepeatCount    int        `json:"repeat_count"`
	CorrectCount   int        `json:"correct_count"`
	IncorrectCount int        `json:"incorrect_count"`
	MasteryScore   float64    `json:"mastery_score"`
	LastSeenAt     *time.Time `json:"last_seen_at"`
	NextDue        *time.Time `json:"next_due"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

type syncPullResponse struct {
	SnapshotVersion      *models.ContentSnapshotVersion `json:"snapshot_version,omitempty"`
	NeedDownloadSnapshot bool                           `json:"need_download_snapshot"`
	Progress             []models.UserWordProgress      `json:"progress"`
}

func SyncPush(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var req syncPushRequest
	if !bindJSON(c, &req) {
		return
	}

	tx := DB.Begin()
	if tx.Error != nil {
		handleDBErr(c, tx.Error)
		return
	}

	resp := syncPushResponse{}
	now := time.Now()

	for _, e := range req.Events {
		var existing models.SyncEvent
		if err := tx.Where("event_id = ?", e.EventID).First(&existing).Error; err == nil {
			resp.Duplicates++
			continue
		}

		syncEvent := models.SyncEvent{
			EventID:          e.EventID,
			UserID:           uid,
			DeviceID:         e.DeviceID,
			EntityType:       e.EntityType,
			EntityID:         e.EntityID,
			EventType:        e.EventType,
			PayloadJSON: 			datatypes.JSON(e.Payload),
			ClientCreatedAt:  e.ClientCreatedAt,
			ServerReceivedAt: now,
			Status:           "received",
		}
		if err := tx.Create(&syncEvent).Error; err != nil {
			tx.Rollback()
			handleDBErr(c, err)
			return
		}

		if err := applySyncEvent(tx, uid, e, now); err != nil {
			_ = tx.Model(&syncEvent).Updates(map[string]any{"status": "rejected"}).Error
			resp.Failed++
			continue
		}

		_ = tx.Model(&syncEvent).Updates(map[string]any{"status": "processed", "processed_at": now}).Error
		resp.Accepted++
	}

	if err := tx.Commit().Error; err != nil {
		handleDBErr(c, err)
		return
	}
	respondOK(c, resp)
}

func SyncPull(c *gin.Context) {
	uid, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	var snapshot models.ContentSnapshotVersion
	err := DB.Where("snapshot_type = ? AND is_active = TRUE", "words_base").Order("version_code DESC").First(&snapshot).Error
	if err != nil && err.Error() != "record not found" {
		handleDBErr(c, err)
		return
	}

	var progress []models.UserWordProgress
	if err := DB.Where("user_id = ?", uid).Order("word_id ASC").Find(&progress).Error; err != nil {
		handleDBErr(c, err)
		return
	}

	resp := syncPullResponse{NeedDownloadSnapshot: snapshot.ID != 0, Progress: progress}
	if snapshot.ID != 0 {
		resp.SnapshotVersion = &snapshot
	}
	respondOK(c, resp)
}

func applySyncEvent(tx *gorm.DB, uid uint64, e syncPushEvent, now time.Time) error {
	switch e.EventType {
	case "attempt":
		var p attemptPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return err
		}
		attemptedAt := now
		if p.AttemptedAt != nil {
			attemptedAt = *p.AttemptedAt
		}
		attempt := models.Attempt{
			UserID:         uid,
			WordID:         p.WordID,
			Result:         p.Result,
			ResponseValue:  p.ResponseValue,
			ResponseTimeMs: p.ResponseTimeMs,
			DeviceID:       e.DeviceID,
			AttemptedAt:    attemptedAt,
			SyncedAt:       now,
		}
		return tx.Create(&attempt).Error

	case "progress_update":
		var p progressPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return err
		}
		updatedAt := now
		if p.UpdatedAt != nil {
			updatedAt = *p.UpdatedAt
		}
		progress := models.UserWordProgress{
			UserID:         uid,
			WordID:         p.WordID,
			Box:            p.Box,
			RepeatCount:    p.RepeatCount,
			CorrectCount:   p.CorrectCount,
			IncorrectCount: p.IncorrectCount,
			MasteryScore:   p.MasteryScore,
			LastSeenAt:     p.LastSeenAt,
			NextDue:        p.NextDue,
			UpdatedAt:      updatedAt,
		}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "word_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"box",
				"repeat_count",
				"correct_count",
				"incorrect_count",
				"mastery_score",
				"last_seen_at",
				"next_due",
				"updated_at",
			}),
		}).Create(&progress).Error

	default:
		return nil
	}
}
