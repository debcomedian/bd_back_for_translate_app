package models

import "time"

type UserWordProgress struct {
	UserID         uint64  `gorm:"primaryKey;autoIncrement:false;index:idx_user_word_progress_user_due;index:idx_user_word_progress_user_box"`
	WordID         uint64  `gorm:"primaryKey;autoIncrement:false"`
	Box            int     `gorm:"not null;default:0"`
	RepeatCount    int     `gorm:"not null;default:0"`
	CorrectCount   int     `gorm:"not null;default:0"`
	IncorrectCount int     `gorm:"not null;default:0"`
	MasteryScore   float64 `gorm:"not null;default:0"`
	LastSeenAt     *time.Time
	NextDue        *time.Time `gorm:"index:idx_user_word_progress_next_due;index:idx_user_word_progress_user_due"`
	UpdatedAt      time.Time  `gorm:"not null;default:now()"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE;"`
	Word Word `gorm:"foreignKey:WordID;references:ID;constraint:OnDelete:CASCADE;"`
}

func (UserWordProgress) TableName() string {
	return "user_word_progress"
}
