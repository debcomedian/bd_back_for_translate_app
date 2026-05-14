package models

import "time"

type User struct {
	ID           uint64    `gorm:"primaryKey"`
	Username     string    `gorm:"type:text;not null;uniqueIndex"`
	Email        string    `gorm:"type:text;not null;uniqueIndex"`
	PasswordHash string    `gorm:"type:text;not null"`
	Level        *string   `gorm:"type:text;default:'A1'"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`
	UpdatedAt    time.Time `gorm:"not null;default:now()"`

	Attempts     []Attempt          `gorm:"foreignKey:UserID"`
	WordProgress []UserWordProgress `gorm:"foreignKey:UserID"`
	SyncEvents   []SyncEvent        `gorm:"foreignKey:UserID"`
}
