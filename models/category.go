package models

import "time"

type Category struct {
	ID        uint64    `gorm:"primaryKey"`
	Slug      string    `gorm:"type:text;not null;uniqueIndex"`
	NameRu    string    `gorm:"type:text;not null"`
	NameEn    string    `gorm:"type:text;not null"`
	NameDe    string    `gorm:"type:text;not null"`
	Entity    string    `gorm:"type:text;not null;default:word"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`

	Words []Word `gorm:"foreignKey:CategoryID"`
}
