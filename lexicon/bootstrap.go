package lexicon

import (
	"log"
	"os"
	"strconv"

	"gorm.io/gorm"
)

func BootstrapContentIfNeeded(db *gorm.DB) error {
	if os.Getenv("AUTO_IMPORT_ACTIVE_BANK") != "true" {
		log.Println("[lexicon bootstrap] AUTO_IMPORT_ACTIVE_BANK=false, skipping content bootstrap")
		return nil
	}

	threshold := envInt("CONTENT_BOOTSTRAP_THRESHOLD", 1000)
	var count int64
	if err := db.Model(&LexicalConcept{}).Where("is_active = TRUE").Count(&count).Error; err != nil {
		return err
	}
	if int(count) >= threshold {
		log.Printf("[lexicon bootstrap] concepts=%d, threshold=%d, skipping import", count, threshold)
		return nil
	}

	path := os.Getenv("ACTIVE_BANK_PATH")
	if path == "" {
		path = "data/lexicon/active_bank.jsonl"
	}
	report, err := ImportActiveBank(db, path)
	if err != nil {
		return err
	}
	log.Printf("[lexicon bootstrap] imported: processed=%d concepts=%d forms=%d directions=%d snapshot=%d", report.Processed, report.Concepts, report.Forms, report.Directions, report.SnapshotVersionCode)
	return nil
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
