package main

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

func ResetTestDatabase(db *gorm.DB, cfg Config) error {
	testDBName, err := databaseNameFromURL(cfg.TestDatabaseURL)
	if err != nil {
		return err
	}
	if err := guardTestDatabaseName(testDBName, cfg); err != nil {
		return err
	}

	query := `
TRUNCATE TABLE
	audit_log,
	sync_events,
	attempts,
	user_word_progress,
	words_meta_lang,
	words_meta_base,
	word_synonyms,
	words,
	categories,
	admin_users,
	users,
	content_snapshot_versions
RESTART IDENTITY CASCADE;
`
	if err := db.Exec(query).Error; err != nil {
		return fmt.Errorf("не удалось очистить тестовые таблицы: %w", err)
	}

	log.Printf("[тестовый контур] тестовые таблицы очищены в базе %s", testDBName)
	return nil
}
