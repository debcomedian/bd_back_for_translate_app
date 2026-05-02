package models

import "time"

type ContentSnapshotVersion struct {
	ID           uint64     `gorm:"primaryKey"`
	SnapshotType string     `gorm:"type:text;not null;uniqueIndex:idx_snapshot_type_version;index:idx_snapshot_type_active"`
	VersionCode  int64      `gorm:"not null;uniqueIndex:idx_snapshot_type_version"`
	Checksum     string     `gorm:"type:text;not null"`
	IsActive     bool       `gorm:"not null;default:false;index:idx_snapshot_type_active"`
	PublishedAt  *time.Time
	CreatedAt    time.Time  `gorm:"not null;default:now()"`
}

func (ContentSnapshotVersion) TableName() string {
	return "content_snapshot_versions"
}