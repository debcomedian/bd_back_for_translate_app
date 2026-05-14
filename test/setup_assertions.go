package main

import (
	"fmt"
	"log"

	"bd_back_for_translate_app/models"

	"gorm.io/gorm"
)

func AssertPreparedTestState(db *gorm.DB, cfg Config) error {
	if !cfg.AssertPreparedState {
		log.Println("[тестовый контур] проверки подготовленного состояния отключены")
		return nil
	}

	if err := AssertSyncEventsSchema(db); err != nil {
		return err
	}
	if err := AssertAdminSeedExists(db, cfg); err != nil {
		return err
	}
	if err := AssertWordOnlyDataPipeline(db, cfg); err != nil {
		return err
	}

	log.Println("[тестовый контур] проверки подготовленного состояния пройдены")
	return nil
}

func AssertSyncEventsSchema(db *gorm.DB) error {
	requiredColumns := []string{
		"event_id",
		"user_id",
		"device_id",
		"entity_type",
		"entity_id",
		"event_type",
		"payload_json",
		"client_created_at",
		"server_received_at",
		"processed_at",
		"status",
	}

	for _, column := range requiredColumns {
		var count int64
		err := db.Raw(`
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'sync_events'
  AND column_name = ?
`, column).Scan(&count).Error
		if err != nil {
			return fmt.Errorf("проверка колонки sync_events.%s: %w", column, err)
		}
		if count != 1 {
			return fmt.Errorf("проверка колонки sync_events.%s: ожидалось 1, получено %d", column, count)
		}
	}

	log.Println("[тестовый контур] проверка схемы sync_events пройдена")
	return nil
}

func AssertAdminSeedExists(db *gorm.DB, cfg Config) error {
	var count int64
	err := db.Model(&models.AdminUser{}).
		Where("username = ? AND email = ? AND role = ? AND is_active = TRUE", cfg.TestAdminUsername, cfg.TestAdminEmail, "admin").
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("проверка наличия тестового администратора: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("проверка тестового администратора: ожидалась 1 активная запись для %s/%s, получено %d", cfg.TestAdminUsername, cfg.TestAdminEmail, count)
	}

	log.Printf("[тестовый контур] проверка тестового администратора пройдена: username=%s email=%s", cfg.TestAdminUsername, cfg.TestAdminEmail)
	return nil
}

func AssertWordOnlyDataPipeline(db *gorm.DB, cfg Config) error {
	var wordsCount int64
	if err := db.Model(&models.Word{}).Where("source_ref LIKE ?", "active_bank:%").Count(&wordsCount).Error; err != nil {
		return fmt.Errorf("проверка количества записей active_bank: %w", err)
	}
	if wordsCount < cfg.MinActiveBankWords {
		return fmt.Errorf("проверка количества записей active_bank: ожидалось >= %d, получено %d", cfg.MinActiveBankWords, wordsCount)
	}

	var metaLangCount int64
	if err := db.Model(&models.WordMetaLang{}).Count(&metaLangCount).Error; err != nil {
		return fmt.Errorf("проверка количества языковых метаданных: %w", err)
	}
	if metaLangCount < cfg.MinMetaLangRows {
		return fmt.Errorf("проверка количества языковых метаданных: ожидалось >= %d, получено %d", cfg.MinMetaLangRows, metaLangCount)
	}

	var metaBaseCount int64
	if err := db.Model(&models.WordMetaBase{}).Count(&metaBaseCount).Error; err != nil {
		return fmt.Errorf("проверка количества базовых метаданных: %w", err)
	}
	if metaBaseCount < cfg.MinMetaBaseRows {
		return fmt.Errorf("проверка количества базовых метаданных: ожидалось >= %d, получено %d", cfg.MinMetaBaseRows, metaBaseCount)
	}
	if metaBaseCount < wordsCount {
		return fmt.Errorf("проверка покрытия базовых метаданных: ожидалось не менее количества записей active_bank %d, получено %d", wordsCount, metaBaseCount)
	}

	var synonymsCount int64
	if err := db.Model(&models.WordSynonym{}).Count(&synonymsCount).Error; err != nil {
		return fmt.Errorf("проверка количества синонимов: %w", err)
	}
	if synonymsCount < cfg.MinSynonymRows {
		return fmt.Errorf("проверка количества синонимов: ожидалось >= %d, получено %d", cfg.MinSynonymRows, synonymsCount)
	}

	var snapshotCount int64
	if err := db.Model(&models.ContentSnapshotVersion{}).
		Where("snapshot_type = ? AND is_active = TRUE", "words_base").
		Count(&snapshotCount).Error; err != nil {
		return fmt.Errorf("проверка активного snapshot words_base: %w", err)
	}
	if snapshotCount != 1 {
		return fmt.Errorf("проверка активного snapshot words_base: ожидался 1 активный snapshot, получено %d", snapshotCount)
	}

	log.Printf(
		"[тестовый контур] проверки словарного пайплайна пройдены: записей_active_bank=%d языковых_метаданных=%d базовых_метаданных=%d синонимов=%d активных_snapshot=%d",
		wordsCount,
		metaLangCount,
		metaBaseCount,
		synonymsCount,
		snapshotCount,
	)
	return nil
}
