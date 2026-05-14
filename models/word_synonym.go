package models

import "time"

type WordSynonym struct {
	ID           uint64    `gorm:"primaryKey"`
	WordID       uint64    `gorm:"not null;index:idx_word_synonyms_word_id;uniqueIndex:idx_word_synonyms_unique"`
	LangCode     string    `gorm:"type:text;not null;uniqueIndex:idx_word_synonyms_unique"`
	SynonymValue string    `gorm:"type:text;not null;uniqueIndex:idx_word_synonyms_unique"`
	IsPrimary    bool      `gorm:"not null;default:false"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`

	Word Word `gorm:"foreignKey:WordID;references:ID;constraint:OnDelete:CASCADE;"`
}

func (WordSynonym) TableName() string {
	return "word_synonyms"
}
