package services

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"bd_back_for_translate_app/models"

	"gorm.io/gorm"
)

const (
	defaultActiveBankPath            = "data/lexicon/active_bank.jsonl"
	defaultContentBootstrapThreshold = 100
)

func BootstrapContentIfNeeded(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("инициализация контента: подключение к базе данных отсутствует")
	}

	enabled := envBool("AUTO_IMPORT_ACTIVE_BANK", false)
	if !enabled {
		log.Println("[инициализация контента] AUTO_IMPORT_ACTIVE_BANK=false, автоматический импорт пропущен")
		return nil
	}

	threshold := envInt("CONTENT_BOOTSTRAP_THRESHOLD", defaultContentBootstrapThreshold)
	activeBankPath := strings.TrimSpace(os.Getenv("ACTIVE_BANK_PATH"))
	if activeBankPath == "" {
		activeBankPath = defaultActiveBankPath
	}

	var wordsCount int64
	if err := db.Model(&models.Word{}).Count(&wordsCount).Error; err != nil {
		return fmt.Errorf("инициализация контента: не удалось посчитать записи словаря: %w", err)
	}

	log.Printf("[инициализация контента] текущих записей словаря=%d, порог=%d", wordsCount, threshold)

	if wordsCount >= int64(threshold) {
		log.Println("[инициализация контента] порог контента уже достигнут, импорт не требуется")
		return nil
	}

	if _, err := os.Stat(activeBankPath); err != nil {
		return fmt.Errorf("инициализация контента: файл active_bank недоступен по пути %s: %w", activeBankPath, err)
	}

	log.Printf("[инициализация контента] импорт контента из %s", activeBankPath)

	report, err := ImportActiveBank(db, activeBankPath)
	if err != nil {
		return fmt.Errorf("инициализация контента: ошибка импорта active_bank: %w", err)
	}

	log.Printf("[инициализация контента] импорт завершён: обработано=%d создано=%d обновлено=%d метаданных_языков=%d базовых_метаданных=%d синонимов=%d snapshot_опубликован=%t версия_snapshot=%d",
		report.Processed,
		report.CreatedWords,
		report.UpdatedWords,
		report.MetaLangUpserts,
		report.MetaBaseUpserts,
		report.SynonymsInserted,
		report.SnapshotPublished,
		report.SnapshotVersionCode,
	)

	return nil
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
