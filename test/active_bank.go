package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"bd_back_for_translate_app/models"
	"bd_back_for_translate_app/services"

	"gorm.io/gorm"
)

func SeedActiveBank(db *gorm.DB, cfg Config) error {
	activeBankPath, err := resolveActiveBankFixturePath(cfg.TestActiveBankPath)
	if err != nil {
		return err
	}

	report, err := services.ImportActiveBank(db, activeBankPath)
	if err != nil {
		return fmt.Errorf("ошибка импорта тестового active_bank: %w", err)
	}

	if report.Processed == 0 {
		return fmt.Errorf("импорт тестового active_bank не обработал ни одной записи")
	}
	if report.CreatedWords+report.UpdatedWords == 0 {
		return fmt.Errorf("импорт тестового active_bank не создал и не обновил словарные записи: %+v", report)
	}
	if report.MetaLangUpserts == 0 {
		return fmt.Errorf("импорт тестового active_bank не обновил языковые метаданные: %+v", report)
	}
	if report.MetaBaseUpserts == 0 {
		return fmt.Errorf("импорт тестового active_bank не обновил базовые метаданные: %+v", report)
	}
	if report.SynonymsInserted == 0 {
		return fmt.Errorf("импорт тестового active_bank не добавил синонимы: %+v", report)
	}
	if !report.SnapshotPublished || report.SnapshotVersionCode == 0 {
		return fmt.Errorf("импорт тестового active_bank не опубликовал snapshot: %+v", report)
	}

	if err := assertActiveBankImportedRows(db); err != nil {
		return err
	}

	log.Printf(
		"[тестовый контур] отчёт active_bank: путь=%s обработано=%d создано=%d обновлено=%d языковых_метаданных=%d базовых_метаданных=%d синонимов=%d версия_snapshot=%d",
		activeBankPath,
		report.Processed,
		report.CreatedWords,
		report.UpdatedWords,
		report.MetaLangUpserts,
		report.MetaBaseUpserts,
		report.SynonymsInserted,
		report.SnapshotVersionCode,
	)

	return nil
}

func resolveActiveBankFixturePath(configuredPath string) (string, error) {
	candidates := make([]string, 0, 4)
	if configuredPath != "" {
		candidates = append(candidates, configuredPath)
	}
	candidates = append(candidates,
		filepath.ToSlash(filepath.Join("test", "fixtures", "active_bank_test.jsonl")),
		filepath.ToSlash(filepath.Join("data", "lexicon", "active_bank_test.jsonl")),
		filepath.ToSlash(filepath.Join("data", "lexicon", "active_bank.jsonl")),
	)

	seen := make(map[string]bool, len(candidates))
	for _, path := range candidates {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		if _, err := os.Stat(path); err == nil {
			if configuredPath != "" && path != configuredPath {
				log.Printf("[тестовый контур] тестовый active_bank не найден по пути %s, используется резервный путь %s", configuredPath, path)
			}
			return path, nil
		}
	}

	return "", fmt.Errorf(
		"тестовый active_bank недоступен; проверенные пути: %v. Укажите TEST_ACTIVE_BANK_PATH=test/fixtures/active_bank_test.jsonl или создайте этот файл",
		candidates,
	)
}

func assertActiveBankImportedRows(db *gorm.DB) error {
	checks := []struct {
		name  string
		query func() (int64, error)
		min   int64
	}{
		{
			name: "записи active_bank",
			query: func() (int64, error) {
				var count int64
				err := db.Model(&models.Word{}).Where("source_ref LIKE ?", "active_bank:%").Count(&count).Error
				return count, err
			},
			min: 1,
		},
		{
			name: "языковые метаданные",
			query: func() (int64, error) {
				var count int64
				err := db.Model(&models.WordMetaLang{}).Count(&count).Error
				return count, err
			},
			min: 3,
		},
		{
			name: "базовые метаданные",
			query: func() (int64, error) {
				var count int64
				err := db.Model(&models.WordMetaBase{}).Count(&count).Error
				return count, err
			},
			min: 1,
		},
		{
			name: "синонимы",
			query: func() (int64, error) {
				var count int64
				err := db.Model(&models.WordSynonym{}).Count(&count).Error
				return count, err
			},
			min: 1,
		},
		{
			name: "активный snapshot",
			query: func() (int64, error) {
				var count int64
				err := db.Model(&models.ContentSnapshotVersion{}).
					Where("snapshot_type = ? AND is_active = TRUE", "words_base").
					Count(&count).Error
				return count, err
			},
			min: 1,
		},
	}

	for _, check := range checks {
		count, err := check.query()
		if err != nil {
			return fmt.Errorf("проверка %s: %w", check.name, err)
		}
		if count < check.min {
			return fmt.Errorf("проверка %s: ожидалось не менее %d строк, получено %d", check.name, check.min, count)
		}
	}

	return nil
}
