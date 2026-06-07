package main

import (
	"log"
	"time"

	"bd_back_for_translate_app/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedAdminUser(db *gorm.DB, cfg Config) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.TestAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()

	return db.Transaction(func(tx *gorm.DB) error {
		var admin models.AdminUser

		result := tx.Where("username = ? OR email = ?", cfg.TestAdminUsername, cfg.TestAdminEmail).Limit(1).Find(&admin)
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			admin = models.AdminUser{
				Username:     cfg.TestAdminUsername,
				Email:        cfg.TestAdminEmail,
				PasswordHash: string(hash),
				Role:         "admin",
				IsActive:     true,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := tx.Create(&admin).Error; err != nil {
				return err
			}
		} else {
			admin.Username = cfg.TestAdminUsername
			admin.Email = cfg.TestAdminEmail
			admin.PasswordHash = string(hash)
			admin.Role = "admin"
			admin.IsActive = true
			admin.UpdatedAt = now
			if err := tx.Save(&admin).Error; err != nil {
				return err
			}
		}

		log.Printf("[тестовый контур] тестовый администратор создан в admin_users: username=%s email=%s", cfg.TestAdminUsername, cfg.TestAdminEmail)
		return nil
	})
}
