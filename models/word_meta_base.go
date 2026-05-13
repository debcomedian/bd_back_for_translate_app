package models

import "time"

type WordMetaBase struct {
	WordID                  uint64    `gorm:"primaryKey"`
	MetaCefrLevel           *string   `gorm:"type:text;index:idx_words_meta_cefr_level"`
	MetaImportanceScore     int       `gorm:"not null;default:0"`
	MetaFreqBucket          int       `gorm:"not null;default:0;index:idx_words_meta_freq_bucket"`
	MetaLengthChars         int       `gorm:"not null;default:0"`
	MetaBaseDifficulty      float64   `gorm:"not null;default:0;index:idx_words_meta_base_difficulty"`
	MetaRequiredStage       *int
	MetaBlocklistFlag       bool      `gorm:"not null;default:false;index:idx_words_meta_blocklist"`
	MetaForcedIntroduceFlag bool      `gorm:"not null;default:false"`
	MetaVersion             int       `gorm:"not null;default:1"`
	CalculatedAt            time.Time `gorm:"not null;default:now()"`

	Word Word `gorm:"foreignKey:WordID;references:ID;constraint:OnDelete:CASCADE;"`
}

func (WordMetaBase) TableName() string {
	return "words_meta_base"
}
