package models

import "time"

type WordMetaLang struct {
	ID                uint64    `gorm:"primaryKey"`
	WordID            uint64    `gorm:"not null;uniqueIndex:idx_words_meta_lang_word_lang"`
	LangCode          string    `gorm:"type:text;not null;uniqueIndex:idx_words_meta_lang_word_lang;index:idx_words_meta_lang_lang_code"`
	Lemma             string    `gorm:"type:text;not null"`
	LemmaChars        int       `gorm:"not null;default:0"`
	TokenCount        int       `gorm:"not null;default:1"`
	ZipfFrequency     float64   `gorm:"not null;default:0"`
	FreqBucket        int       `gorm:"not null;default:0;index:idx_words_meta_lang_freq_bucket"`
	OrthographyScore  float64   `gorm:"not null;default:0"`
	MultiwordScore    float64   `gorm:"not null;default:0"`
	POSScore          float64   `gorm:"not null;default:0"`
	ConfidencePenalty float64   `gorm:"not null;default:0"`
	LangScore         float64   `gorm:"not null;default:0;index:idx_words_meta_lang_lang_score"`
	CalculatedAt      time.Time `gorm:"not null;default:now()"`

	Word Word `gorm:"foreignKey:WordID;references:ID;constraint:OnDelete:CASCADE;"`
}

func (WordMetaLang) TableName() string {
	return "words_meta_lang"
}
