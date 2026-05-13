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
	defaultActiveBankPath          = "data/lexicon/active_bank.jsonl"
	defaultContentBootstrapThreshold = 100
)

func BootstrapContentIfNeeded(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("bootstrap content: db is nil")
	}

	enabled := envBool("AUTO_IMPORT_ACTIVE_BANK", false)
	if !enabled {
		log.Println("[bootstrap] AUTO_IMPORT_ACTIVE_BANK=false, skipping content bootstrap")
		return nil
	}

	threshold := envInt("CONTENT_BOOTSTRAP_THRESHOLD", defaultContentBootstrapThreshold)
	activeBankPath := strings.TrimSpace(os.Getenv("ACTIVE_BANK_PATH"))
	if activeBankPath == "" {
		activeBankPath = defaultActiveBankPath
	}

	var wordsCount int64
	if err := db.Model(&models.Word{}).Count(&wordsCount).Error; err != nil {
		return fmt.Errorf("bootstrap content: count words: %w", err)
	}

	log.Printf("[bootstrap] current words count=%d, threshold=%d", wordsCount, threshold)

	if wordsCount >= int64(threshold) {
		log.Println("[bootstrap] content threshold already satisfied, skipping import")
		return nil
	}

	if _, err := os.Stat(activeBankPath); err != nil {
		return fmt.Errorf("bootstrap content: active bank file not available at %s: %w", activeBankPath, err)
	}

	log.Printf("[bootstrap] importing content from %s", activeBankPath)

	report, err := ImportActiveBank(db, activeBankPath)
	if err != nil {
		return fmt.Errorf("bootstrap content: import active bank: %w", err)
	}

	log.Printf("[bootstrap] import done: processed=%d created=%d updated=%d meta_lang=%d meta_base=%d synonyms=%d snapshot_published=%t snapshot_version=%d",
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
