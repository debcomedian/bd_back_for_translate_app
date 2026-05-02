package models

import "time"

type Attempt struct {
	ID             uint64    `gorm:"primaryKey"`
	UserID         uint64    `gorm:"not null;index:idx_attempts_user_id;index:idx_attempts_user_word;index:idx_attempts_user_attempted_at"`
	WordID         uint64    `gorm:"not null;index:idx_attempts_word_id;index:idx_attempts_user_word"`
	Result         string    `gorm:"type:text;not null"`
	ResponseValue  *string   `gorm:"type:text"`
	ResponseTimeMs int       `gorm:"not null;default:0"`
	DeviceID       string    `gorm:"type:text;not null"`
	AttemptedAt    time.Time `gorm:"not null;index:idx_attempts_attempted_at;index:idx_attempts_user_attempted_at"`
	SyncedAt       time.Time `gorm:"not null;default:now()"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE;"`
	Word Word `gorm:"foreignKey:WordID;references:ID;constraint:OnDelete:CASCADE;"`
}