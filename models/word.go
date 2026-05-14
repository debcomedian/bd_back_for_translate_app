package models

import "time"

type Word struct {
	ID              uint64    `gorm:"primaryKey"`
	LangCode        string    `gorm:"type:text;not null;index:idx_words_lang_code;index:idx_words_lang_active"`
	WordRu          *string   `gorm:"type:text"`
	WordEn          *string   `gorm:"type:text"`
	WordDe          *string   `gorm:"type:text"`
	TranscriptionRu *string   `gorm:"type:text"`
	TranscriptionEn *string   `gorm:"type:text"`
	TranscriptionDe *string   `gorm:"type:text"`
	ExampleRu       *string   `gorm:"type:text"`
	ExampleEn       *string   `gorm:"type:text"`
	ExampleDe       *string   `gorm:"type:text"`
	SourceRef       *string   `gorm:"type:text"`
	CategoryID      *uint64   `gorm:"index:idx_words_category_id"`
	IsActive        bool      `gorm:"not null;default:true;index:idx_words_lang_active"`
	CreatedAt       time.Time `gorm:"not null;default:now()"`
	UpdatedAt       time.Time `gorm:"not null;default:now()"`

	Category *Category          `gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL;"`
	MetaBase *WordMetaBase      `gorm:"foreignKey:WordID;constraint:OnDelete:CASCADE;"`
	MetaLang []WordMetaLang     `gorm:"foreignKey:WordID;constraint:OnDelete:CASCADE;"`
	Synonyms []WordSynonym      `gorm:"foreignKey:WordID;constraint:OnDelete:CASCADE;"`
	Attempts []Attempt          `gorm:"foreignKey:WordID;constraint:OnDelete:CASCADE;"`
	Progress []UserWordProgress `gorm:"foreignKey:WordID;constraint:OnDelete:CASCADE;"`
}
