package models

import "time"

type AdminUser struct {
	ID           uint64    `gorm:"primaryKey"`
	Username     string    `gorm:"type:text;not null;uniqueIndex"`
	Email        string    `gorm:"type:text;not null;uniqueIndex"`
	PasswordHash string    `gorm:"type:text;not null"`
	Role         string    `gorm:"type:text;not null;default:'editor'"`
	IsActive     bool      `gorm:"not null;default:true"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`
	UpdatedAt    time.Time `gorm:"not null;default:now()"`

	AuditLogs []AuditLog `gorm:"foreignKey:AdminUserID"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}