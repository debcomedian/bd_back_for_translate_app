package models

import (
	"time"

	"gorm.io/datatypes"
)

type SyncEvent struct {
	ID               uint64 `gorm:"primaryKey"`
	EventID          string `gorm:"type:text;not null;uniqueIndex"`
	UserID           uint64 `gorm:"not null;index:idx_sync_events_user_id;index:idx_sync_events_user_received"`
	DeviceID         string `gorm:"type:text;not null;index:idx_sync_events_device_id"`
	EntityType       string `gorm:"type:text;not null"`
	EntityID         *uint64
	EventType        string         `gorm:"type:text;not null"`
	PayloadJSON      datatypes.JSON `gorm:"type:jsonb;not null"`
	ClientCreatedAt  time.Time      `gorm:"not null"`
	ServerReceivedAt time.Time      `gorm:"not null;default:now();index:idx_sync_events_user_received"`
	ProcessedAt      *time.Time
	Status           string `gorm:"type:text;not null;default:'received';index:idx_sync_events_status"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE;"`
}

func (SyncEvent) TableName() string {
	return "sync_events"
}
