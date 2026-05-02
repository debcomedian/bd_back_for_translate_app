package models

import (
	"time"

	"gorm.io/datatypes"
)

type AuditLog struct {
	ID          uint64         `gorm:"primaryKey"`
	AdminUserID uint64         `gorm:"not null;index:idx_audit_log_admin_user_id"`
	ActionType  string         `gorm:"type:text;not null"`
	EntityType  string         `gorm:"type:text;not null;index:idx_audit_log_entity"`
	EntityID    *uint64        `gorm:"index:idx_audit_log_entity"`
	PayloadJSON datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time      `gorm:"not null;default:now();index:idx_audit_log_created_at"`

	AdminUser AdminUser `gorm:"foreignKey:AdminUserID;references:ID;constraint:OnDelete:CASCADE;"`
}

func (AuditLog) TableName() string {
	return "audit_log"
}