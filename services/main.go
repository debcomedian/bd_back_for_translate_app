package services

import (
	"encoding/json"
	"log"
	"os"

	"bd_back_for_translate_app/database"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются переменные окружения системы")
	}

	inputPath := os.Getenv("ACTIVE_BANK_PATH")
	if inputPath == "" {
		inputPath = "data/lexicon/active_bank.jsonl"
	}

	database.Init()

	report, err := ImportActiveBank(database.DB, inputPath)
	if err != nil {
		log.Fatalf("Ошибка импорта active_bank: %v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		log.Fatalf("Ошибка кодирования отчёта: %v", err)
	}
}
